// Package plugin_rewritebody a plugin to rewrite response body.
package plugin_rewritebody

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

// Rewrite holds one rewrite body configuration.
type Rewrite struct {
	Regex       string `json:"regex,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

// Config holds the plugin configuration.
type Config struct {
	LastModified bool      `json:"lastModified,omitempty"`
	Rewrites     []Rewrite `json:"rewrites,omitempty"`
	UseNonce     bool      `json:"useNonce,omitempty"`
}

// CreateConfig creates and initializes the plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

type rewrite struct {
	regex       *regexp.Regexp
	replacement []byte
}

type rewriteBody struct {
	name         string
	next         http.Handler
	rewrites     []rewrite
	lastModified bool
	useNonce     bool
	nonce        string
}

type responseWriter struct {
	buffer       bytes.Buffer
	lastModified bool
	wroteHeader  bool
	statusCode   int

	header   http.Header
	nonce    string
	useNonce bool

	http.ResponseWriter
}

// New creates and returns a new rewrite body plugin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	rewrites := make([]rewrite, len(config.Rewrites))

	for i, rewriteConfig := range config.Rewrites {
		regex, err := regexp.Compile(rewriteConfig.Regex)
		if err != nil {
			return nil, fmt.Errorf("error compiling regex %q: %w", rewriteConfig.Regex, err)
		}

		rewrites[i] = rewrite{
			regex:       regex,
			replacement: []byte(rewriteConfig.Replacement),
		}
	}

	var nonce string
	if config.UseNonce {
		nonce = generateNonce()
	}

	return &rewriteBody{
		name:         name,
		next:         next,
		rewrites:     rewrites,
		lastModified: config.LastModified,
		useNonce:     config.UseNonce,
		nonce:        nonce,
	}, nil
}

func (r *rewriteBody) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrappedWriter := &responseWriter{
		lastModified:   r.lastModified,
		ResponseWriter: rw,
		header:         cloneHeader(rw.Header()),
		nonce:          r.nonce,
		useNonce:       r.useNonce,
	}

	r.next.ServeHTTP(wrappedWriter, req)

	bodyBytes := wrappedWriter.buffer.Bytes()

	contentEncoding := wrappedWriter.Header().Get("Content-Encoding")
	if contentEncoding != "" && contentEncoding != "identity" {
		wrappedWriter.commitTo(rw, bodyBytes)
		return
	}

	// Check if response is HTML before rewriting
	contentType := wrappedWriter.Header().Get("Content-Type")
	if !isHTMLContent(contentType) {
		wrappedWriter.commitTo(rw, bodyBytes)
		return
	}

	for _, rwt := range r.rewrites {
		replacement := rwt.replacement
		bodyBytes = rwt.regex.ReplaceAll(bodyBytes, replacement)
	}

	// Update CSP header with nonce if enabled and add nonce to all script tags
	if r.useNonce {
		wrappedWriter.updateCSPWithNonce()
		bodyBytes = wrappedWriter.addNonceToScriptTags(bodyBytes)
	}

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

func (r *responseWriter) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
}

func (r *responseWriter) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}

	if !r.lastModified {
		r.Header().Del("Last-Modified")
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
