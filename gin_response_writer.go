package moesifgin

import (
	"bufio"
	"bytes"
	"log"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	noWritten     = -1
	defaultStatus = http.StatusOK
)

// This wraps the gin.ResponseWriter to capture the response body for logging
type logGinResponseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	size   int
	status int
}

var _ gin.ResponseWriter = (*logGinResponseWriter)(nil) // Ensure that it implements the interface completely

func NewLogGinResponseWriter(w gin.ResponseWriter) *logGinResponseWriter {
	return &logGinResponseWriter{
		ResponseWriter: w,
		body:           new(bytes.Buffer),
		size:           noWritten,
		status:         defaultStatus,
	}
}

func (w *logGinResponseWriter) WriteHeader(code int) {
	if w.status != code {
		if w.Written() {
			log.Printf("[WARNING] Headers were already written. Wanted to override status code %d with %d", w.status, code)
			return
		}
		w.status = code
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *logGinResponseWriter) Write(data []byte) (int, error) {
	w.WriteHeaderNow()           // Ensure header is written if it's not already
	n, err := w.body.Write(data) // Write to the buffer first
	if err != nil {
		return n, err
	}
	return w.ResponseWriter.Write(data) // Write to the underlying ResponseWriter
}

func (w *logGinResponseWriter) WriteString(s string) (int, error) {
	w.WriteHeaderNow()
	n, err := w.body.WriteString(s) // Write string to the buffer
	if err != nil {
		return n, err
	}
	return w.ResponseWriter.Write([]byte(s)) // Convert string to bytes and write
}

func (w *logGinResponseWriter) Body() *bytes.Buffer {
	return w.body
}

func (w *logGinResponseWriter) Flush() {
	w.WriteHeaderNow()
	w.ResponseWriter.(http.Flusher).Flush()
	if w.body.Len() > 0 {
		log.Printf("Buffered response body: %s", w.body.String()) // Log or process the buffered response body
	}
}

func (w *logGinResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *logGinResponseWriter) WriteHeaderNow() {
	if !w.Written() {
		w.size = 0
		w.ResponseWriter.WriteHeader(w.status)
	}
}

func (w *logGinResponseWriter) Status() int {
	return w.status
}

func (w *logGinResponseWriter) Size() int {
	return w.size
}

func (w *logGinResponseWriter) Written() bool {
	return w.size != noWritten
}

// Hijack implements the http.Hijacker interface.
func (w *logGinResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.size < 0 {
		w.size = 0
	}
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

// CloseNotify implements the http.CloseNotifier interface.
func (w *logGinResponseWriter) CloseNotify() <-chan bool {
	return w.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

func (w *logGinResponseWriter) Pusher() (pusher http.Pusher) {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher
	}
	return nil
}
