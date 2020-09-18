package handlerchain

import (
	"net/http"
)

type bufferingHttpResponseWriter struct {
	http.ResponseWriter
	buffer []byte
}

func (w *bufferingHttpResponseWriter) Write(b []byte) (int, error) {
	if w.buffer == nil {
		w.buffer = make([]byte, 0, len(b))
	}
	w.buffer = append(w.buffer, b...)
	return w.ResponseWriter.Write(b)
}

func (w *bufferingHttpResponseWriter) Buffer() []byte {
	return w.buffer
}