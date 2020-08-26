package handlerchain

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

type BufferedHttpResponseWriter interface {
	Content() []byte
}

type BufferingHttpResponseWriter struct {
	http.ResponseWriter
	buffer []byte
}

func (w BufferingHttpResponseWriter) Write(b []byte) (int, error) {
	w.buffer = append(w.buffer, b...)
	return w.ResponseWriter.Write(b)
}

func (w BufferingHttpResponseWriter) Content() []byte {
	return w.buffer
}

type LinkableHandler http.HandlerFunc

type PreHandler func(w http.ResponseWriter, r *http.Request, cancelFunc context.CancelFunc)

type PostHandler func(w http.ResponseWriter, r *http.Request)

type PrepFunc func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request)

func Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handling")
}

func Auth(w http.ResponseWriter, r *http.Request, cancelFunc context.CancelFunc) {
	fmt.Println("authing")
	if r.Header.Get("Authorization") == "" {
		w.WriteHeader(403)
		fmt.Println("canceling no AUTH")
		cancelFunc()
	}
}

func JustGet(w http.ResponseWriter, r *http.Request, cancelFunc context.CancelFunc) {
	fmt.Println("just getting")
	if r.Method != http.MethodGet {
		fmt.Println("canceling no GET")
		cancelFunc()
	}
}

func ResponseLogger(w http.ResponseWriter, r *http.Request) {
	fmt.Println("post logger")
	switch v := w.(type) {
	case BufferedHttpResponseWriter:
		fmt.Println(string(v.Content()))
	default:
		fmt.Println("cannot log response body, call LinkableHandler.bufferResponse at the end of the handler chain to fix this.")
	}
}

func main() {
	http.HandleFunc("/a", toLinkable(Handler).Before(JustGet, Auth).After(ResponseLogger).bufferResponse())
	http.HandleFunc("/b", ChainBuilder{}.
		For(Handler).
		Before(JustGet, Auth).
		After(ResponseLogger).
		Prepare(func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
			return BufferingHttpResponseWriter{
				ResponseWriter: w,
				buffer:         make([]byte, 0),
			}, r
		}).
		Handle())

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func toLinkable(h http.HandlerFunc) LinkableHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r)
	}
}

func (h PreHandler) toHandlerFunc() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx, cancel := context.WithCancel(request.Context())
		cancelableRequest := request.Clone(ctx)
		h(writer, cancelableRequest, cancel)
	}
}

func (base LinkableHandler) Before(handlerFunc ...PreHandler, ) LinkableHandler {
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

func (base LinkableHandler) After(handlerFunc ...PostHandler) LinkableHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		base(w, r)
		for _, handler := range handlerFunc {
			handler(w, r)
		}
	}
}

func (base LinkableHandler) bufferResponse() LinkableHandler {
	return func(writer http.ResponseWriter, request *http.Request) {
		bw := BufferingHttpResponseWriter{
			ResponseWriter: writer,
			buffer:         make([]byte, 0),
		}
		base(bw, request)
	}
}

type ChainBuilder struct {
	post []PostHandler
	pre  []PreHandler
	main http.HandlerFunc
	prep []PrepFunc
}

func (c ChainBuilder) For(h http.HandlerFunc) ChainBuilder {
	c.main = h
	return c
}

func (c ChainBuilder) Before(handlers ...PreHandler) ChainBuilder {
	if c.pre == nil {
		c.pre = make([]PreHandler, 0, len(handlers))
	}

	c.pre = append(handlers, c.pre...)

	return c
}

func (c ChainBuilder) Prepare(prepFuncs ...PrepFunc) ChainBuilder {
	if c.prep == nil {
		c.prep = make([]PrepFunc, 0, len(prepFuncs))
	}

	c.prep = append(c.prep, prepFuncs...)

	return c
}

func (c ChainBuilder) After(handlers ...PostHandler) ChainBuilder {
	if c.post == nil {
		c.post = make([]PostHandler, 0, len(handlers))
	}
	c.post = append(c.post, handlers...)
	return c
}

func (c ChainBuilder) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wp := w
		rp := r
		for _, p := range c.prep {
			wp, rp = p(wp, rp)
		}

		if len(c.pre) > 0 {
			ctx, cancel := context.WithCancel(r.Context())
			cancelableRequest := r.Clone(ctx)
			rp = cancelableRequest
			for _, pre := range c.pre {
				pre(wp, rp, cancel)
			}
		}

		c.main(wp, rp)

		for _, post := range c.post {
			post(wp, rp)
		}

	}
}
