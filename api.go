// handlerchain is a package for adding pre- and post-processing to http.HandleFuncs
package handlerchain

import (
	"context"
	"net/http"
)


// ERR_UNBUFFERED, a handler wanted to read the responseBody, but it was not buffered
const ERR_UNBUFFERED errstring = "buffered response body required"

// ERR_UNBUFFERED, a handler wanted to cancel the whole request, but it was not cancelable
const ERR_NOTCANCELABLE errstring = "root request has to be cancelable"

// PreHandler execute some logic, before calling the next handler in the chain.
// calling cancelFunc will prevent any later PreHandlers or the main Handler from running.
// if the handlers are added by `PreWithRunPost`, the PostHandlers will still be executed
// if the handlers are added by `PreWithCancelPost`, the PostHandlers will NOT be executed
// the later requires the root request to be cancelable (call `Cancelable()` when building the chain)
type PreHandler func(w http.ResponseWriter, r *http.Request, cancelFunc context.CancelFunc)

// PostHandler execute some logic, after the main Handler has returned
// If you need to access the written ResponseBody, make sure to use a `BufferedPostHandler` instead
type PostHandler func(w http.ResponseWriter, r *http.Request)

// BufferedPostHandler execute some logic, after the main Handler has returned
// If you need to access the written ResponseBody, make sure to call Buffered when building the Chain
type BufferedPostHandler func(w BufferedWriter, r *http.Request)

// PrepFunc are executed before any handler executes
// They are used to prepare the Request or ResponseWriter, eg. to make re ResponseWriter buffer the response or manipulate the
// Requests context.
type PrepFunc func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request)


// HandleChain takes your actual HandlerFunc and starts building the HandlerChain.
// This enables pre- and post=processing of requests and responses
func HandleChain(h http.HandlerFunc) linkableHandler {
	return linkableHandler(h)
}

// BufferedWriter is the interface to access buffered Responses in PostHandlers
type BufferedWriter interface {
	http.ResponseWriter
	Buffer() []byte
}

// ExpectBufferedWriter is a utility to ensure a writer is buffered for a PostHandler that needs to access the responseBody
func ExpectBufferedWriter(w http.ResponseWriter) (BufferedWriter, error) {
	wb, ok := w.(BufferedWriter)
	if !ok {
		return nil, ERR_UNBUFFERED
	}
	return wb, nil
}