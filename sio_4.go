package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	sio4 "github.com/zishang520/socket.io/socket"
)

// -=-=-=-=-=-=-=- sio.v4
type sio4wrapper struct {
	sio       *sio4.Server
	serveHTTP http.Handler
}

func (x *sio4wrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	x.serveHTTP.ServeHTTP(w, r)
}
func (x *sio4wrapper) Close() {
	x.sio.Close(nil)
}
func SetupSocketIoV4() (x *sio4wrapper) {
	sio := sio4.NewServer(nil, nil)
	x = &sio4wrapper{
		sio:       sio,
		serveHTTP: sio.ServeHandler(nil),
	}

	emitEvent2 := NewEmitSocket()
	x.sio.On("connection", func(clients ...any) {
		client := clients[0].(*sio4.Socket)
		Log.Info("connection socket", "id", client.Id(), "remote", client.Handshake().Address)
		emitEvent2.AddKeyValue(client.Id(), client)

		client.On("event", func(datas ...any) {
			fmt.Println("event from ", client)
			emitEvent2.AddKeyValue(client.Id(), client)
		})
		client.On("disconnect", func(datas ...any) {
			fmt.Println("disconnected from ", datas[0])
			emitEvent2.RemoveKeyValue(client.Id())
			Log.Info("on disconnect relase hijack block ", "error", SyncHijackMap.Release(client.Handshake().Address))
		})

		client.On("error", func(a ...any) {
			fmt.Println("error on ", a[0])

		})

	})
	go func() {
		c := 0
		for {

			<-time.After(time.Second)
			for _, v := range emitEvent2.LoadFunc()() {
				v.Emit("event", fmt.Sprintf("%d - event msg", c))
				v.Emit("hello", fmt.Sprintf("%d - hello msg", c))
			}
			c += 1
		}
	}()
	return
}

// *-*-*
type SocketMapFunc func() map[sio4.SocketId]*sio4.Socket

var emptySocketMapFunc = func() map[sio4.SocketId]*sio4.Socket { return map[sio4.SocketId]*sio4.Socket{} }

type EmitSocket struct {
	atomic.Pointer[SocketMapFunc]
}

func (e *EmitSocket) StoreFunc(f SocketMapFunc) {
	e.Store(&f)
}
func (e *EmitSocket) LoadFunc() SocketMapFunc {
	return *e.Load()
}
func (e *EmitSocket) AddKeyValue(k sio4.SocketId, v *sio4.Socket) {
	m := e.LoadFunc()()
	m[k] = v
	e.StoreFunc(func() map[sio4.SocketId]*sio4.Socket { return m })
}
func (e *EmitSocket) RemoveKeyValue(k sio4.SocketId) {
	m := e.LoadFunc()()
	delete(m, k)
	e.StoreFunc(func() map[sio4.SocketId]*sio4.Socket { return m })
}

func NewEmitSocket() *EmitSocket {
	r := &EmitSocket{Pointer: atomic.Pointer[SocketMapFunc]{}}
	r.StoreFunc(emptySocketMapFunc)
	return r
}

// *-*-*
