package tunneloverquic

import (
	"github.com/quic-go/quic-go"
	"sync"
)

type SessionManager struct {
	sync.Mutex
	clients map[string][]*quic.Connection
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		clients: map[string][]*quic.Connection{},
	}
}

func (sm *SessionManager) Add() {

}

func (sm *SessionManager) Remove() {

}
