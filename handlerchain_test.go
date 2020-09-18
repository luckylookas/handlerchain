package handlerchain

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

type MockLoggingResponseWriter struct {
	log []byte
}

func (w *MockLoggingResponseWriter) Write(b []byte) (int, error) {
	w.log = append(w.log, b...)
	return len(b), nil
}

func (w *MockLoggingResponseWriter) Header() http.Header {
	return nil
}

func (w *MockLoggingResponseWriter) WriteHeader(i int) {

}

func preCancel(w http.ResponseWriter, r *http.Request, cancelFunc context.CancelFunc) {
	w.Write([]byte("CANCEL"))
	cancelFunc()
}

func preA(w http.ResponseWriter, r *http.Request, cancelFunc context.CancelFunc) {
	w.Write([]byte("PreA-"))
}

func preB(w http.ResponseWriter, r *http.Request, cancelFunc context.CancelFunc) {
	w.Write([]byte("PreB-"))

}

func postA(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("-PostA"))

}

func postB(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("-PostB"))

}

func prepA(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	w.Write([]byte("PrepA-"))
	return w, r
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Handler"))

}

func postExpectBuffered(w BufferedWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprint("--", len(w.Buffer()))))
}

func Test_Cancel_doesNotCancelPostHandlers(t *testing.T) {
	assertions := assert.New(t)

	// All Handlers append to the response body for assertions, they have to be called in this sequence for the string equals assertion to pass
	handleFunc := HandleChain(handler).PreWithRunPost(preA, preCancel).Post(postA, postB).Prepare(prepA).Cancelable()
	w := &MockLoggingResponseWriter{
		log: []byte{},
	}
	req, err := http.NewRequest("GET", "", nil)
	assertions.NoError(err)

	handleFunc(w, req)
	assertions.Equal("PrepA-PreA-CANCEL-PostA-PostB", string(w.log))
}


func Test_RootCancel_doesCancelPostHandlers(t *testing.T) {
	assertions := assert.New(t)

	// All Handlers append to the response body for assertions, they have to be called in this sequence for the string equals assertion to pass
	handleFunc := HandleChain(handler).PreWithCancelPost(preA, preCancel).Post(postA, postB).Cancelable()
	w := &MockLoggingResponseWriter{
		log: []byte{},
	}
	req, err := http.NewRequest("GET", "", nil)
	assertions.NoError(err)

	handleFunc(w, req)
	assertions.Equal("PreA-CANCEL", string(w.log))
}

func Test_Chain(t *testing.T) {
	assertions := assert.New(t)

	// All Handlers append to the response body for assertions, they have to be called in this sequence for the string equals assertion to pass
	handleFunc := HandleChain(handler).PreWithRunPost(preA, preB).Post(postA, postB).PostBuffered(postExpectBuffered).Prepare(prepA).Buffered()
	w := &MockLoggingResponseWriter{
		log: []byte{},
	}
	req, err := http.NewRequest("GET", "", nil)
	assertions.NoError(err)

	handleFunc(w, req)
	assertions.Equal("PrepA-PreA-PreB-Handler-PostA-PostB--35", string(w.log))
}