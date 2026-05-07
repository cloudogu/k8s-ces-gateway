// Package plugin_nonce adds nonces to inline script tags
package plugin_nonce

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"regexp"
	"strings"
)

// Config holds the plugin configuration.
type Config struct {
}

// CreateConfig creates and initializes the plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

type nonce struct {
	next http.Handler
}

type responseWriter struct {
	buffer       bytes.Buffer
	wroteHeader  bool
	statusCode   int
	nonce        string

	header       http.Header
	originalRw   http.ResponseWriter
	passthrough  bool
}

// New creates and returns a new nonce plugin instance.
func New(_ context.Context, next http.Handler, _ *Config, name string) (http.Handler, error) {
	return &nonce{
		next: next,
	}, nil
}

func (n *nonce) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrappedWriter := &responseWriter{
		originalRw: rw,
		nonce:          generateNonce(),
		header:         cloneHeader(rw.Header()),
	}

	n.next.ServeHTTP(wrappedWriter, req)

	if !wrappedWriter.wroteHeader {
		wrappedWriter.WriteHeader(http.StatusOK)
	}

	if wrappedWriter.passthrough {
		return
	}

	bodyBytes := wrappedWriter.buffer.Bytes()

	bodyBytes = wrappedWriter.addNonceToScriptTags(bodyBytes)
	wrappedWriter.updateCSPWithNonce()

	wrappedWriter.commit(bodyBytes)
}

// generateNonce creates a cryptographically secure random nonce
func generateNonce() string {
	b := make([]byte, 16) // 128 bit
	// rand.Read cannot throw an error per specification
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// updateCSPWithNonce adds the nonce to existing CSP header
func (r *responseWriter) updateCSPWithNonce() {
	header := r.Header()
	csp := header.Get("Content-Security-Policy")
	if csp == "" {
		// there should be a csp header
		return
	}

	nonceValue := fmt.Sprintf("'nonce-%s'", r.nonce)
	updatedCSP := strings.Replace(csp, "script-src ", "script-src "+nonceValue+" ", 1)

	header.Set("Content-Security-Policy", updatedCSP)
}

// addNonceToScriptTags adds nonce attribute to all <script> tags that don't already have one
func (r *responseWriter) addNonceToScriptTags(body []byte) []byte {
	scriptRegex := regexp.MustCompile(`<script(\s+[^>]*)?>`)

	result := scriptRegex.ReplaceAllFunc(body, func(match []byte) []byte {
		matchStr := string(match)

		// Remove existing nonce attribute if present
		nonceAttrRegex := regexp.MustCompile(`\s+nonce="[^"]*"`)
		matchStr = nonceAttrRegex.ReplaceAllString(matchStr, "")

		if strings.HasSuffix(matchStr, ">") {
			return []byte(fmt.Sprintf(`<script nonce="%s"%s`, r.nonce, strings.TrimPrefix(matchStr, "<script")))
		}

		return match
	})

	return result
}

func (r *responseWriter) Header() http.Header {
	if r.passthrough {
		return r.originalRw.Header()
	}

	if r.header == nil {
		r.header = make(http.Header)
	}

	return r.header
}

func (r *responseWriter) isContentReplacable() bool {
	contentType := r.Header().Get("Content-Type")
	mtype, _, _ := mime.ParseMediaType(contentType)

	if strings.ToLower(mtype) != "text/html" {
		return false
	}

	encoding := r.Header().Get("Content-Encoding")
	if encoding == "" {
		return true
	}

	ctype, _, err := mime.ParseMediaType(encoding)
	if err != nil {
		return false
	}

	return ctype == "" || strings.ToLower(ctype) == "identity"
}

func (r *responseWriter) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true

	if r.passthrough {
		r.originalRw.WriteHeader(statusCode)
		return
	}

	r.statusCode = statusCode

	if !r.isContentReplacable() {
		r.passthrough = true
		applyHeader(r.originalRw.Header(), r.header)
		r.originalRw.WriteHeader(statusCode)
	}
}

func (r *responseWriter) Write(data []byte) (n int, err error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	if r.passthrough {
		return r.originalRw.Write(data)
	}

	return r.buffer.Write(data)
}

func (r *responseWriter) commit(body []byte) {
	status := r.statusCode
	if status == 0 {
		status = http.StatusOK
	}

	r.header.Del("Content-Length")
	applyHeader(r.originalRw.Header(), r.header)
	r.originalRw.WriteHeader(status)

	if _, err := r.originalRw.Write(body); err != nil {
		log.Printf("unable to write body: %v", err)
	}
}
func cloneHeader(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for k, vv := range src {
		cpy := make([]string, len(vv))
		copy(cpy, vv)
		dst[k] = cpy
	}
	return dst
}

func applyHeader(dst, src http.Header) {
	for k := range dst {
		dst.Del(k)
	}
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.originalRw.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T is not a http.Hijacker", r.originalRw)
	}

	return hijacker.Hijack()
}

func (r *responseWriter) Flush() {
	if flusher, ok := r.originalRw.(http.Flusher); ok {
		flusher.Flush()
	}
}
