package tunneloverquic

import (
	"context"
	"github.com/quic-go/quic-go"
	"io"
	"net"
)

func clientDial(str quic.Stream, message *Message) error {
	conn, err := net.Dial(message.Proto, message.Address)
	if err != nil {
		return err
	}
	defer conn.Close()

	pipe(str, conn)

	return nil
}
func pipe(client quic.Stream, server net.Conn) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer func() {
		_ = server.Close()
	}()

	go func(cancelFunc context.CancelFunc) {
		_, _ = io.Copy(server, client)
		cancelFunc()
	}(cancel)

	_, _ = io.Copy(client, server)
	cancel()
	<-ctx.Done()
}
