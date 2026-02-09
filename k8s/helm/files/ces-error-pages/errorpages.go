// Package plugin_ces_error_pages redirects to error pages
package plugin_ces_error_pages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
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
		ResponseWriter: rw,
		statusCode:     http.StatusOK,
	}

	s.next.ServeHTTP(wrappedWriter, req)

	statusCode := wrappedWriter.statusCode

	// Check if this status code should be handled
	if !s.statusCodes[statusCode] {
		s.writeResponse(rw, wrappedWriter, statusCode)
		return
	}

	contentType := wrappedWriter.Header().Get("Content-Type")
	bodyBytes := wrappedWriter.buffer.Bytes()

	// only redirect to error page if the content type is HTML and the body is empty
	// some dogus send incorrect status codes with valid html bodies, these should not be redirected
	if s.isHTMLContent(contentType) && s.isEmptyBody(bodyBytes) {
		log.Printf("[CesErrorPages] Redirecting status %d with empty HTML body to error page", statusCode)
		s.redirectToErrorPage(rw, req, statusCode)
		return
	}

	// write original response
	s.writeResponse(rw, wrappedWriter, statusCode)
}

func (s *cesErrorPages) writeResponse(rw http.ResponseWriter, wrappedWriter *responseWriter, statusCode int) {
	for key, values := range wrappedWriter.Header() {
		// content-length is set automatically
		if key == "Content-Length" {
			continue
		}
		rw.Header().Del(key)
		for _, value := range values {
			rw.Header().Add(key, value)
		}
	}

	// Write status code
	rw.WriteHeader(statusCode)

	// Write body
	if _, err := rw.Write(wrappedWriter.buffer.Bytes()); err != nil {
		log.Printf("[CesErrorPages] unable to write body: %v", err)
	}
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

// isHTMLContent checks for text/html content type (with or without charset)
func (s *cesErrorPages) isHTMLContent(contentType string) bool {
	if contentType == "" {
		return false
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	return strings.HasPrefix(contentType, "text/html")
}

// isEmptyBody checks if body is empty or contains only whitespace
func (s *cesErrorPages) isEmptyBody(body []byte) bool {
	return len(bytes.TrimSpace(body)) == 0
}

type responseWriter struct {
	buffer     bytes.Buffer
	statusCode int

	http.ResponseWriter
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	// Don't call ResponseWriter.WriteHeader yet - we need to buffer everything first
}

func (r *responseWriter) Write(p []byte) (int, error) {
	return r.buffer.Write(p)
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
