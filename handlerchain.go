package handlerchain

import (
	"context"
	"log"
	"net/http"
)

type linkableHandler http.HandlerFunc
type preplinkableHandler http.HandlerFunc

// PreWithRunPost adds PreHandlers that, when the request is canceled by them, will prevent the main handler from running but still run all PostHandlers
func (base linkableHandler) PreWithRunPost(handlerFunc ...PreHandler, ) linkableHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		cancelableRequest := r.Clone(ctx)
		for _, handler := range handlerFunc {
			handler(w, cancelableRequest, cancel)
			if cancelableRequest.Context().Err() != nil {
				return
			}
		}
		base(w, cancelableRequest)
	}
}

// PreWithCancelPost adds PreHandlers that, when the request is canceled by them, will cancel the entire chain.
func (base linkableHandler) PreWithCancelPost(handlerFunc ...PreHandler, ) linkableHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		rootcancel := r.Context().Value("rootCancel")
		if (rootcancel == nil) {
			w.WriteHeader(500)
			log.Fatalln("%w", ERR_NOTCANCELABLE)
		}
		rootCancelFunc, ok := rootcancel.(context.CancelFunc)
		if !ok {
			w.WriteHeader(500)
			log.Fatalln("%w", ERR_NOTCANCELABLE)
		}
		for _, handler := range handlerFunc {
			handler(w, r, rootCancelFunc)
			if r.Context().Err() != nil {
				return
			}
		}
		base(w, r)
	}
}

// Post will add PostHandlers that do NOT need to access the ResponseBody
func (base linkableHandler) Post(handlerFunc ...PostHandler) linkableHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		base(w, r)
		if r.Context().Err() != nil {
			return
		}
		for _, handler := range handlerFunc {
			handler(w, r)
		}
	}
}

// PostBuffered will add PostHandlers that do need to access the ResponseBody
func (base linkableHandler) PostBuffered(handlerFunc ...BufferedPostHandler) linkableHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		base(w, r)
		if r.Context().Err() != nil {
			return
		}
		buffered, err := ExpectBufferedWriter(w)
		if err != nil {
			log.Fatalln(ERR_UNBUFFERED)
			return
		}
		for _, handler := range handlerFunc {
			handler(buffered, r)
		}
	}
}



// todo unify these

// Buffered will buffer the ResponseBody to make it accessible to PostHandlers
func (base linkableHandler) Buffered() preplinkableHandler {
	return preplinkableHandler(getBufferFunc(http.HandlerFunc(base)))
}

// Cancelable will enable canceling the entire chain from any handler
func (base linkableHandler) Cancelable() preplinkableHandler {
	return preplinkableHandler(getCancelableFunc(http.HandlerFunc(base)))
}

// Prepare enables preparing the Request and Response (eg. add Headers, lock Headers, buffer response Body)
func (base linkableHandler) Prepare(handlerFunc ...PrepFunc) preplinkableHandler {
	return preplinkableHandler(getPrepFunc(http.HandlerFunc(base), handlerFunc...))
}

// Buffered will buffer the ResponseBody to make it accessible to PostHandlers
func (base preplinkableHandler) Buffered() preplinkableHandler {
	return preplinkableHandler(getBufferFunc(http.HandlerFunc(base)))
}

// Cancelable will enable canceling the entire chain from any handler
func (base preplinkableHandler) Cancelable() preplinkableHandler {
	return preplinkableHandler(getCancelableFunc(http.HandlerFunc(base)))
}

// Prepare enables preparing the Request and Response (eg. add Headers, lock Headers, buffer response Body)
func (base preplinkableHandler) Prepare(handlerFunc ...PrepFunc) preplinkableHandler {
	return preplinkableHandler(getPrepFunc(http.HandlerFunc(base), handlerFunc...))
}

func getCancelableFunc(base http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		ctx = context.WithValue(ctx, "rootCancel", cancel)
		req := r.WithContext(ctx)
		base(w, req)
	}
}

func getBufferFunc(base http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		bw := &bufferingHttpResponseWriter{
			ResponseWriter: writer,
			buffer:         make([]byte, 0),
		}
		base(bw, request)
	}
}

func getPrepFunc(base http.HandlerFunc, handlerFunc ...PrepFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		prepW := writer
		prepR := request
		for _, handler := range handlerFunc {
			prepW, prepR = handler(prepW, prepR)
		}
		base(prepW, prepR)
	}
}