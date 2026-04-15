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
	buffer      bytes.Buffer
	wroteHeader bool
	nonce       string
	statusCode  int

	header http.Header
	http.ResponseWriter
}

// New creates and returns a new nonce plugin instance.
func New(_ context.Context, next http.Handler, _ *Config, name string) (http.Handler, error) {
	return &nonce{
		next: next,
	}, nil
}

func (n *nonce) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrappedWriter := &responseWriter{
		ResponseWriter: rw,
		nonce:          generateNonce(),
		header:         cloneHeader(rw.Header()),
	}

	n.next.ServeHTTP(wrappedWriter, req)
	bodyBytes := wrappedWriter.buffer.Bytes()

	contentEncoding := wrappedWriter.Header().Get("Content-Encoding")
	if contentEncoding != "" && contentEncoding != "identity" {
		wrappedWriter.commitTo(rw, bodyBytes)
		return
	}

	// Check if response is HTML before adding nonces
	contentType := wrappedWriter.Header().Get("Content-Type")
	if !isHTMLContent(contentType) {
		wrappedWriter.commitTo(rw, bodyBytes)
		return
	}

	bodyBytes = wrappedWriter.addNonceToScriptTags(bodyBytes)
	wrappedWriter.updateCSPWithNonce()

	wrappedWriter.commitTo(rw, bodyBytes)
}

// generateNonce creates a cryptographically secure random nonce
func generateNonce() string {
	b := make([]byte, 16) // 128 bit
	if _, err := rand.Read(b); err != nil {
		log.Printf("failed to generate nonce: %v", err)
		return ""
	}
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

func (r *responseWriter) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}

	r.wroteHeader = true
	r.statusCode = statusCode

	// Delegates the Content-Length Header creation to the final body write.
	r.Header().Del("Content-Length")
}

func (r *responseWriter) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	return r.buffer.Write(p)
}

func (r *responseWriter) commitTo(rw http.ResponseWriter, body []byte) {
	status := r.statusCode
	if status == 0 {
		status = http.StatusOK
	}

	r.Header().Del("Content-Length")
	applyHeader(rw.Header(), r.Header())
	rw.WriteHeader(status)

	if _, err := rw.Write(body); err != nil {
		log.Printf("unable to write body: %v", err)
	}
}

func (r *responseWriter) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
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
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T is not a http.Hijacker", r.ResponseWriter)
	}

	return hijacker.Hijack()
}

func (r *responseWriter) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// isHTMLContent checks if the content type indicates HTML
func isHTMLContent(contentType string) bool {
	if contentType == "" {
		return false
	}

	// Check for text/html (with or without charset)
	return len(contentType) >= 9 && contentType[:9] == "text/html"
}
