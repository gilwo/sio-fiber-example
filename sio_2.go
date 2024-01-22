package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	sio2 "github.com/googollee/go-socket.io"
	sio2engine "github.com/googollee/go-socket.io/engineio"
)

// -=-=-=-=-=-=-=- sio.v1/2
type sio2wrapper struct {
	sio       *sio2.Server
	serveHTTP http.Handler
}

func (x *sio2wrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	x.serveHTTP.ServeHTTP(w, r)
}
func (x *sio2wrapper) Close() {
	x.sio.Close()
}
func SetupSocketIoV2() *sio2wrapper {
	so := sio2.NewServer(&sio2engine.Options{})
	{
		roomParticipants := map[string]sio2.Conn{}
		// connect and create room
		so.OnConnect("/", func(s sio2.Conn) error {
			s.SetContext("")
			s.Join("bilbao")
			Log.Info("new connection", "socket id", s.ID(), "remote", s.RemoteAddr())
			roomParticipants[s.ID()] = s
			return nil
		})

		// chat in room
		so.OnEvent("/", "chat", func(s sio2.Conn, msg string) {
			so.BroadcastToRoom("/", "bilbao", "chat", msg)
		})

		// message from the admin
		go func() {
			for {
				<-time.After(3 * time.Second)
				so.BroadcastToRoom("/", "bilbao", "chat",
					fmt.Sprintf("admin message (%d participatns)", len(roomParticipants)))
			}
		}()

		// some other examples
		so.OnEvent("/", "notice", func(s sio2.Conn, msg string) {
			Log.Info("new notice", "socket id", s.ID(), "message", msg)
			s.Emit("reply", "have "+msg)
		})
		so.OnEvent("/chat", "msg", func(s sio2.Conn, msg string) string {
			s.SetContext(msg)
			Log.Info("new message", "socket id", s.ID(), "message", msg)
			return "recv " + msg
		})

		so.OnEvent("/", "bye", func(s sio2.Conn) string {
			Log.Info("bye message", "socket id", s.ID())
			last := s.Context().(string)
			s.Emit("bye", last)
			s.Close()
			return last
		})

		so.OnError("/", func(s sio2.Conn, e error) {
			Log.Error(e, "error on socket namespace", "socket id", s.ID())
		})

		so.OnDisconnect("/", func(s sio2.Conn, reason string) {
			Log.Info("disconnect message", "socket id", s.ID())
			delete(roomParticipants, s.ID())
		})

		go func() {
			err := so.Serve()
			if err != nil && err != io.EOF {
				Log.Error(err, "sio2 server start failed")
			}
		}()
	}
	return &sio2wrapper{
		sio:       so,
		serveHTTP: so,
	}
}
