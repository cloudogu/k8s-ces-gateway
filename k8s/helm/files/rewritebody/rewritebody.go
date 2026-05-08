// Package plugin_rewritebody a plugin to rewrite response body.
package plugin_rewritebody

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"mime"
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
}

type responseWriter struct {
	buffer       bytes.Buffer
	lastModified bool
	wroteHeader  bool
	statusCode   int

	header      http.Header
	originalRw  http.ResponseWriter
	passthrough bool
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

	return &rewriteBody{
		name:         name,
		next:         next,
		rewrites:     rewrites,
		lastModified: config.LastModified,
	}, nil
}

func (r *rewriteBody) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrappedWriter := &responseWriter{
		lastModified: r.lastModified,
		originalRw:   rw,
		header:       cloneHeader(rw.Header()),
	}

	r.next.ServeHTTP(wrappedWriter, req)

	if !wrappedWriter.wroteHeader {
		wrappedWriter.WriteHeader(http.StatusOK)
	}

	if wrappedWriter.passthrough {
		return
	}

	bodyBytes := wrappedWriter.buffer.Bytes()

	for _, rwt := range r.rewrites {
		bodyBytes = rwt.regex.ReplaceAll(bodyBytes, rwt.replacement)
	}

	wrappedWriter.commit(bodyBytes)
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

	if r.isContentReplacable() {
		if !r.lastModified {
			r.header.Del("Last-Modified")
		}
	} else {
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
