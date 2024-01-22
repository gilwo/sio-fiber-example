package main

import (
	"errors"
	"sync"
)

type SyncHijack struct {
	sync.Map
}

type SyncBlock chan any

func (h *SyncHijack) Block(handle interface{}) SyncBlock {
	ch := make(SyncBlock)
	h.Store(handle, ch)
	return ch
}
func (h *SyncHijack) Release(handle interface{}) error {
	_ch, ok := h.Load(handle)
	if ok {
		close(_ch.(SyncBlock))
		h.Delete(handle)
		return nil
	}
	return errors.New("syncblock not found for handle")
}

var SyncHijackMap SyncHijack
