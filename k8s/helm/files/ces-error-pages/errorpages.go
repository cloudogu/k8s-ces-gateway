// Package plugin_ces_error_pages redirects to error pages
package plugin_ces_error_pages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Config holds the plugin configuration.
type Config struct {
	ErrorPageService string `json:"errorPageService,omitempty"`
	ErrorPagePort    int    `json:"errorPagePort,omitempty"`
	ErrorPagePath    string `json:"errorPagePath,omitempty"`
	StatusCodes      []int  `json:"statusCodes,omitempty"`
}

// CreateConfig this default config will be overridden by the plugin config
func CreateConfig() *Config {
	return &Config{}
}

type cesErrorPages struct {
	name             string
	next             http.Handler
	errorPageService string
	errorPagePort    int
	errorPagePath    string
	statusCodes      map[int]bool
	httpClient       *http.Client
}

// New creates and returns a new ces error pages plugin instance
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.ErrorPagePath == "" {
		return nil, fmt.Errorf("errorPagePath cannot be empty")
	}

	statusCodesMap := make(map[int]bool, len(config.StatusCodes))
	for _, code := range config.StatusCodes {
		statusCodesMap[code] = true
	}

	log.Printf("[CesErrorPages] Plugin initialized with status codes: %v", config.StatusCodes)

	return &cesErrorPages{
		name:             name,
		next:             next,
		errorPageService: config.ErrorPageService,
		errorPagePort:    config.ErrorPagePort,
		errorPagePath:    config.ErrorPagePath,
		statusCodes:      statusCodesMap,
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // Prevent hanging requests
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}, nil
}

func (s *cesErrorPages) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrappedWriter := &responseWriter{
		originalRw: rw,
		statusCode: http.StatusOK,
		header:       cloneHeader(rw.Header()),
	}

	s.next.ServeHTTP(wrappedWriter, req)

	if !wrappedWriter.wroteHeader {
		wrappedWriter.WriteHeader(http.StatusOK)
	}

	if wrappedWriter.passthrough {
		return
	}

	statusCode := wrappedWriter.statusCode

	bodyBytes := wrappedWriter.buffer.Bytes()

	// Check if this status code should be handled
	if !s.statusCodes[statusCode] {
		wrappedWriter.commit(bodyBytes)
		return
	}

	// only redirect to error page if the content type is HTML and the body is empty
	// some dogus send incorrect status codes with valid html bodies, these should not be redirected
	if s.isEmptyBody(bodyBytes) {
		log.Printf("[CesErrorPages] Redirecting status %d with empty HTML body to error page", statusCode)
		s.redirectToErrorPage(rw, req, statusCode)
		return
	}

	// write original response
	wrappedWriter.commit(bodyBytes)
}

// redirectToErrorPage redirects the client to the error page for the given status code
func (s *cesErrorPages) redirectToErrorPage(rw http.ResponseWriter, req *http.Request, statusCode int) {
	errorPath := strings.ReplaceAll(s.errorPagePath, "{status}", strconv.Itoa(statusCode))
	errorURL := fmt.Sprintf("http://%s:%d%s", s.errorPageService, s.errorPagePort, errorPath)

	errorReq, err := http.NewRequestWithContext(req.Context(), http.MethodGet, errorURL, nil)
	if err != nil {
		log.Printf("[CesErrorPages] error creating error page request: %v", err)
		http.Error(rw, http.StatusText(statusCode), statusCode)
		return
	}

	// Forward only relevant headers
	if userAgent := req.Header.Get("User-Agent"); userAgent != "" {
		errorReq.Header.Set("User-Agent", userAgent)
	}
	if acceptLang := req.Header.Get("Accept-Language"); acceptLang != "" {
		errorReq.Header.Set("Accept-Language", acceptLang)
	}

	resp, err := s.httpClient.Do(errorReq)
	if err != nil {
		log.Printf("[CesErrorPages] error fetching error page: %v", err)
		http.Error(rw, http.StatusText(statusCode), statusCode)
		return
	}
	defer resp.Body.Close()

	// Copy error page response headers
	for key, values := range resp.Header {
		// these are set automatically
		if key == "Content-Length" || key == "Transfer-Encoding" {
			continue
		}
		rw.Header().Del(key)
		for _, value := range values {
			rw.Header().Add(key, value)
		}
	}

	// Write the original error status code
	rw.WriteHeader(statusCode)

	if _, err := io.Copy(rw, resp.Body); err != nil {
		log.Printf("[CesErrorPages] error writing error page body: %v", err)
	}
}

// isEmptyBody checks if body is empty or contains only whitespace
func (s *cesErrorPages) isEmptyBody(body []byte) bool {
	return len(bytes.TrimSpace(body)) == 0
}

type responseWriter struct {
	buffer      bytes.Buffer
	wroteHeader bool
	statusCode  int

	header      http.Header
	originalRw  http.ResponseWriter
	passthrough bool
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
