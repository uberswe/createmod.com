package server

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

// RequestEvent is a drop-in replacement for pocketbase/core.RequestEvent.
// It wraps the standard http.ResponseWriter and *http.Request and provides
// the same helper methods (HTML, JSON, String, Redirect, Stream, etc.)
// so that handler code needs minimal changes.
type RequestEvent struct {
	Response http.ResponseWriter
	Request  *http.Request

	written bool
	status  int
}

// NewRequestEvent creates a RequestEvent from standard HTTP primitives.
func NewRequestEvent(w http.ResponseWriter, r *http.Request) *RequestEvent {
	return &RequestEvent{
		Response: w,
		Request:  r,
	}
}

// --- Response helpers (matching PocketBase's API) ---

// HTML writes an HTML response.
func (e *RequestEvent) HTML(status int, data string) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.WriteHeader(status)
	_, err := e.Response.Write([]byte(data))
	e.written = true
	e.status = status
	return err
}

// String writes a plain-text response.
func (e *RequestEvent) String(status int, data string) error {
	e.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	e.Response.WriteHeader(status)
	_, err := e.Response.Write([]byte(data))
	e.written = true
	e.status = status
	return err
}

// JSON writes a JSON response.
func (e *RequestEvent) JSON(status int, data any) error {
	e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	e.Response.WriteHeader(status)
	e.written = true
	e.status = status
	return json.NewEncoder(e.Response).Encode(data)
}

// XML writes an XML response.
func (e *RequestEvent) XML(status int, data any) error {
	e.Response.Header().Set("Content-Type", "application/xml; charset=utf-8")
	e.Response.WriteHeader(status)
	_, _ = e.Response.Write([]byte(xml.Header))
	e.written = true
	e.status = status
	return xml.NewEncoder(e.Response).Encode(data)
}

// Blob writes raw bytes with the given content type.
func (e *RequestEvent) Blob(status int, contentType string, b []byte) error {
	e.Response.Header().Set("Content-Type", contentType)
	e.Response.WriteHeader(status)
	_, err := e.Response.Write(b)
	e.written = true
	e.status = status
	return err
}

// Stream writes from a reader with the given content type.
func (e *RequestEvent) Stream(status int, contentType string, reader io.Reader) error {
	e.Response.Header().Set("Content-Type", contentType)
	e.Response.WriteHeader(status)
	_, err := io.Copy(e.Response, reader)
	e.written = true
	e.status = status
	return err
}

// NoContent writes an empty response with the given status code.
func (e *RequestEvent) NoContent(status int) error {
	e.Response.WriteHeader(status)
	e.written = true
	e.status = status
	return nil
}

// Redirect sends an HTTP redirect.
func (e *RequestEvent) Redirect(status int, url string) error {
	http.Redirect(e.Response, e.Request, url, status)
	e.written = true
	e.status = status
	return nil
}

// FileFS serves a file from the given filesystem.
func (e *RequestEvent) FileFS(fsys fs.FS, filename string) error {
	http.ServeFileFS(e.Response, e.Request, fsys, filename)
	e.written = true
	return nil
}

// Written returns true if a response has already been written.
func (e *RequestEvent) Written() bool {
	return e.written
}

// Status returns the response status code (0 if not yet written).
func (e *RequestEvent) Status() int {
	return e.status
}

// SetCookie adds a Set-Cookie header to the response.
func (e *RequestEvent) SetCookie(cookie *http.Cookie) {
	http.SetCookie(e.Response, cookie)
}

// Flush flushes buffered data to the client if the ResponseWriter supports it.
func (e *RequestEvent) Flush() error {
	if f, ok := e.Response.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

// --- Request inspection helpers ---

// RealIP extracts the client IP from common proxy headers, falling back
// to RemoteAddr.
func (e *RequestEvent) RealIP() string {
	if ip := e.Request.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := e.Request.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if xff := e.Request.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(e.Request.RemoteAddr)
	if err != nil {
		return e.Request.RemoteAddr
	}
	return host
}

// IsTLS reports whether the connection uses TLS (directly or via proxy).
func (e *RequestEvent) IsTLS() bool {
	if e.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https")
}

// --- Error helpers ---

// NotFoundError returns a 404 error.
func (e *RequestEvent) NotFoundError(message string, data any) error {
	if message == "" {
		message = "Not Found"
	}
	return &APIError{Status: http.StatusNotFound, Message: message}
}

// BadRequestError returns a 400 error.
func (e *RequestEvent) BadRequestError(message string, data any) error {
	if message == "" {
		message = "Bad Request"
	}
	return &APIError{Status: http.StatusBadRequest, Message: message}
}

// UnauthorizedError returns a 401 error.
func (e *RequestEvent) UnauthorizedError(message string, data any) error {
	if message == "" {
		message = "Unauthorized"
	}
	return &APIError{Status: http.StatusUnauthorized, Message: message}
}

// ForbiddenError returns a 403 error.
func (e *RequestEvent) ForbiddenError(message string, data any) error {
	if message == "" {
		message = "Forbidden"
	}
	return &APIError{Status: http.StatusForbidden, Message: message}
}

// InternalServerError returns a 500 error.
func (e *RequestEvent) InternalServerError(message string, data any) error {
	if message == "" {
		message = "Internal Server Error"
	}
	return &APIError{Status: http.StatusInternalServerError, Message: message}
}


// BindBody reads the request body and binds it to the given struct.
// It supports JSON (application/json) and URL-encoded form data.
// The struct should use `json` or `form` tags for field mapping.
func (e *RequestEvent) BindBody(dst any) error {
	ct := e.Request.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		return json.NewDecoder(e.Request.Body).Decode(dst)
	}
	// Form data: parse and use reflection to map form fields
	if err := e.Request.ParseForm(); err != nil {
		return err
	}
	return bindForm(e.Request.Form, dst)
}

// --- Supporting types ---

// APIError is an error with an HTTP status code.
type APIError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// bindForm maps url.Values to a struct using form tags, falling back to json tags.
func bindForm(values url.Values, dst any) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("bindForm: expected non-nil pointer, got %T", dst)
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("bindForm: expected pointer to struct, got pointer to %s", v.Kind())
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("form")
		if tag == "" {
			tag = field.Tag.Get("json")
		}
		if tag == "" || tag == "-" {
			continue
		}
		val := values.Get(tag)
		if val == "" {
			continue
		}
		fv := v.Field(i)
		if fv.CanSet() && fv.Kind() == reflect.String {
			fv.SetString(val)
		}
	}
	return nil
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%d: %s", e.Status, e.Message)
}
