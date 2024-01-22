package main

import (
	"bufio"
	"io"
	golog "log"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// *!*!*!*!*!*!*!*!*

func CustomNewFastHTTPHandlerFunc(h http.HandlerFunc, hijackHandler ...func(net.Conn)) fasthttp.RequestHandler {
	return CustomNewFastHTTPHandler(h, hijackHandler...)
}
func CustomNewFastHTTPHandler(h http.Handler, hijackHandler ...func(net.Conn)) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var r http.Request
		if err := fasthttpadaptor.ConvertRequest(ctx, &r, true); err != nil {
			ctx.Logger().Printf("cannot parse requestURI %q: %v", r.RequestURI, err)
			ctx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
			return
		}
		w := NewNetHttpResponseWriter(ctx, hijackHandler...)
		h.ServeHTTP(w, r.WithContext(ctx))

		ctx.SetStatusCode(w.StatusCode())
		haveContentType := false
		for k, vv := range w.Header() {
			if k == fasthttp.HeaderContentType {
				haveContentType = true
			}

			for _, v := range vv {
				ctx.Response.Header.Add(k, v)
			}
		}
		if !haveContentType {
			// From net/http.ResponseWriter.Write:
			// If the Header does not contain a Content-Type line, Write adds a Content-Type set
			// to the result of passing the initial 512 bytes of written data to DetectContentType.
			l := 512
			b := ctx.Response.Body()
			if len(b) < 512 {
				l = len(b)
			}
			ctx.Response.Header.Set(fasthttp.HeaderContentType, http.DetectContentType(b[:l]))
		}
	}
}

type netHTTPResponseWriter struct {
	statusCode    int
	h             http.Header
	w             io.Writer
	r             io.Reader
	conn          net.Conn
	ctx           *fasthttp.RequestCtx
	hijackHandler []func(net.Conn)
}

func NewNetHttpResponseWriter(ctx *fasthttp.RequestCtx, hijackHandler ...func(net.Conn)) *netHTTPResponseWriter {
	return &netHTTPResponseWriter{
		w:             ctx.Response.BodyWriter(),
		r:             ctx.RequestBodyStream(),
		conn:          ctx.Conn(),
		ctx:           ctx,
		hijackHandler: hijackHandler,
	}
}

func (w *netHTTPResponseWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func (w *netHTTPResponseWriter) Header() http.Header {
	if w.h == nil {
		w.h = make(http.Header)
	}
	return w.h
}

func (w *netHTTPResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *netHTTPResponseWriter) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

func (w *netHTTPResponseWriter) Flush() {}

type wrapperConn struct {
	net.Conn
	wg   sync.WaitGroup
	once sync.Once
}

func (c *wrapperConn) Close() (err error) {
	dlog.Output(2, "connection closed called")
	c.once.Do(func() {
		defer c.wg.Done()
		err = c.Conn.Close()
		_ = atomic.AddInt32(&ActiveHijackedConnection, -1)
	})
	return
}

func (w *netHTTPResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn := &wrapperConn{Conn: w.conn}
	conn.wg.Add(1)

	w.ctx.HijackSetNoResponse(true)
	w.ctx.Hijack(func(c net.Conn) {
		if len(w.hijackHandler) > 0 {
			w.hijackHandler[0](c)
			go conn.Close()
		}
		dlog.Output(2, "connection waiting")
		conn.wg.Wait()
	})
	_ = atomic.AddInt32(&ActiveHijackedConnection, 1)
	return conn, &bufio.ReadWriter{Reader: bufio.NewReader(w.r), Writer: bufio.NewWriter(w.w)}, nil
}

var dlog *golog.Logger
var ActiveHijackedConnection int32

func init() {
	dlog = golog.New(os.Stdout, "fasthttp: ", golog.LstdFlags|golog.Ltime|golog.Llongfile|golog.Lmsgprefix)
}
