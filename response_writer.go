package floki

import (
	//"bufio"
	//"fmt"
	//"net"
	"net/http"
)

type (
	// ResponseWriter is a wrapper around http.ResponseWriter that provides extra information about
	// the response. It is recommended that middleware handlers use this construct to wrap a responsewriter
	// if the functionality calls for it.
	ResponseWriter interface {
		http.ResponseWriter
		http.Flusher
		// Status returns the status code of the response or 0 if the response has not been written.
		Status() int
		// Written returns whether or not the ResponseWriter has been written.
		Written() bool

		reset(http.ResponseWriter)
		setStatus(int)
	}

	responseWriter struct {
		http.ResponseWriter
		status  int
		written bool
	}
)

func (w *responseWriter) reset(writer http.ResponseWriter) {
	w.ResponseWriter = writer
	w.status = 0
	w.written = false
}

func (w *responseWriter) setStatus(code int) {
	w.status = code
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Written() bool {
	return w.written
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) CloseNotify() <-chan bool {
	return rw.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

func (rw *responseWriter) Flush() {
	flusher, ok := rw.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}
