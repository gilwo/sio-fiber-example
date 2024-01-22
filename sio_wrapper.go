package main

import "net/http"

// -=-=-=-=-=-=-=- sio wrapper
type wrapper interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	Close()
}
