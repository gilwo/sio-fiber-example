package main

import (
	"fmt"
	"log"
	"os"
)

// -=-=-=-=-=-=-=-

type mlog struct {
	*log.Logger
}

func (m *mlog) _prep(v ...interface{}) string {
	l := ""
	for _, e := range v {
		l += fmt.Sprintf("%v", e)
	}
	return l
}
func (m *mlog) Info(v ...interface{}) {
	m.Output(2, "Info:"+m._prep(v))
}
func (m *mlog) Error(v ...interface{}) {
	m.Output(2, "Error:"+m._prep(v))
}

func SetupLog(prefix string) {
	Log = mlog{log.New(os.Stdout, "socket.io with fasthttp: ", log.LstdFlags|log.Ltime|log.Lshortfile|log.Lmsgprefix)}
}

var Log mlog
