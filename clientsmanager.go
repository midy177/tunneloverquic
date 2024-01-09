package tunneloverquic

import (
	"context"
	"fmt"
	"github.com/quic-go/quic-go"
	"net"
	"sync"
)

type clientsManager struct {
	sync.Mutex
	clients map[string][]quic.Connection
}

func newClientsManager() *clientsManager {
	return &clientsManager{
		clients: map[string][]quic.Connection{},
	}
}

func (sm *clientsManager) Add(clientKey string, conn quic.Connection) {
	sm.Lock()
	defer sm.Unlock()
	sm.clients[clientKey] = append(sm.clients[clientKey], conn)
}

func (sm *clientsManager) Remove(clientKey string) {
	sm.Lock()
	defer sm.Unlock()
	fmt.Println("remove the client of: " + clientKey)
	for _, v := range sm.clients[clientKey] {
		err := v.CloseWithError(200, "the server actively removes the connection.")
		if err != nil {
			// TODO print logger
			continue
		}
	}
	delete(sm.clients, clientKey)
}

func (sm *clientsManager) getDialer(clientKey string) (Dialer, error) {
	sm.Lock()
	defer sm.Unlock()

	clients := sm.clients[clientKey]
	if len(clients) > 0 {
		// TODO
		return func(ctx context.Context, network, address string) (net.Conn, error) {
			str, err := clients[0].OpenStreamSync(ctx)
			if err != nil {
				return nil, err
			}
			return newConnection(str, network, address)
		}, nil
	}

	return nil, fmt.Errorf("failed to find Session for client %s", clientKey)
}
