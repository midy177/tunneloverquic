package tunneloverquic

import (
	"context"
	"net"
)

type Dialer func(ctx context.Context, network, address string) (net.Conn, error)

func (s *Server) HasSession(clientKey string) bool {
	_, err := s.clients.getDialer(clientKey)
	return err == nil
}

func (s *Server) Dialer(clientKey string) Dialer {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		d, err := s.clients.getDialer(clientKey)
		if err != nil {
			return nil, err
		}

		return d(ctx, network, address)
	}
}
