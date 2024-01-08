package tunneloverquic

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"net"
	"sync"
)

type SessionManager struct {
	sync.Mutex
	clients map[string][]quic.Connection
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		clients: map[string][]quic.Connection{},
	}
}

func (sm *SessionManager) Add(clientKey string, conn quic.Connection) {
	sm.Lock()
	defer sm.Unlock()
	sm.clients[clientKey] = append(sm.clients[clientKey], conn)
}

func (sm *SessionManager) Remove(clientKey string) {
	sm.Lock()
	defer sm.Unlock()

	for _, v := range sm.clients[clientKey] {
		err := v.CloseWithError(200, "the server actively removes the connection.")
		if err != nil {
			// TODO print logger
			continue
		}
	}
	delete(sm.clients, clientKey)
}

func (sm *SessionManager) getDialer(clientKey string) (Dialer, error) {
	sm.Lock()
	defer sm.Unlock()

	sessions := sm.clients[clientKey]
	if len(sessions) > 0 {
		// TODO
		return func(ctx context.Context, network, address string) (net.Conn, error) {
			return nil, nil
		}, nil
	}

	return nil, fmt.Errorf("failed to find Session for client %s", clientKey)
}
