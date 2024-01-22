package main

import (
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
)

var (
	handlerFunc = CustomNewFastHTTPHandlerFunc
)

func init() {
	SetupLog("socket.io with fasthttp:")
}

// var useFastHttpHandlerFunc bool

// func init() {
// 	useFastHttpHandlerFunc = true
// 	if useFastHttpHandlerFunc {
// 		handlerFunc = fasthttpadaptor.NewFastHTTPHandlerFunc
// 	}
// }

func main() {
	os.Setenv("DEBUG", "socket.io*")

	sio4 := SetupSocketIoV4()
	sio2 := SetupSocketIoV2()
	app := fiber.New(fiber.Config{})
	app.Use(fiberlogger.New(fiberlogger.ConfigDefault))
	app.Get("/socket.io/*", func(c *fiber.Ctx) error {
		Log.Info("enter", "active hijacked http connection", ActiveHijackedConnection)
		defer func() {
			Log.Info("leave", "active hijacked http connection", ActiveHijackedConnection)
		}()
		Log.Info("query", string(c.OriginalURL()))
		var sio wrapper = sio2
		if version := c.Query("EIO", ""); version == "4" { //|| version == "3" { // weird thing about postman - it claim for v2 in the setting of socket.io but it sends EIO=3 ... ?!?
			sio = sio4
		}

		handlerFunc(
			sio.ServeHTTP,
			func(c net.Conn) {
				Log.Info("hijacked handler called", "remote", c.RemoteAddr())
				<-SyncHijackMap.Block(c.RemoteAddr().String())
				Log.Info("hijacked handler ended")
			},
		)(c.Context())
		return nil

	})
	go func() {
		if err := app.Listen(":3005"); err != nil {
			panic(err)
		}
	}()

	go func() {
		netMux2 := &http.ServeMux{}
		netMux2.Handle("/socket.io/", sio2.serveHTTP)
		go http.ListenAndServe("0.0.0.0:3006", netMux2)
	}()
	go func() {
		netMux4 := &http.ServeMux{}
		netMux4.Handle("/socket.io/", sio4.serveHTTP)
		go http.ListenAndServe("0.0.0.0:3007", netMux4)
	}()
	go func() {
		for {
			<-time.After(time.Second * 3)
			Log.Info("** ", "active hijacked connection: ", ActiveHijackedConnection)
		}
	}()

	exit := make(chan struct{})
	SignalC := make(chan os.Signal, 1)

	signal.Notify(SignalC, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range SignalC {
			switch s {
			case os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				close(exit)
				return
			}
		}
	}()

	<-exit
	sio2.Close()
	sio4.Close()
	Log.Info("socket io wrapper closed")
	os.Exit(0)
}
