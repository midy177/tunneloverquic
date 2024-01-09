package tunneloverquic

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"time"
)

func ClientConnect(addr string, auth []byte, tlsConfig TLSConfigurator, quicConfig QuicConfigurator) (*ConnectHandle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 3s handshake timeout
	defer cancel()
	if tlsConfig == nil {
		tlsConfig = func() *tls.Config {
			return &tls.Config{
				// 在生产环境中请配置适当的证书和密钥
				InsecureSkipVerify: true,
			}
		}
	}
	conn, err := quic.DialAddr(ctx, addr, tlsConfig(), quicConfig())
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	streamHandle := &ConnectHandle{
		Conn: conn,
	}

	go clientPing(conn)
	go streamHandle.ConnHandle(false, func(msg []byte) (clientKey string, authed bool, err error) {
		return "local", true, err
	})
	return streamHandle, nil
}

func clientAuth(conn quic.Connection, auth []byte) {
	str, err := conn.OpenStream()
	if err != nil {
		return
	}
	defer func(str quic.Stream) {
		_ = str.Close()
		_ = conn.CloseWithError(200, "sending a ping message expecting a pong reply, but the reply is not a pong message.")
	}(str)
	msg := NewAuthMessage(int64(str.StreamID()), auth)
	_, err = msg.WriteTo(str)
	if err != nil {
		return
	}
	buf := make([]byte, 1024)
	n, err := str.Read(buf)
	if err != nil {
		return
	}
}

func clientPing(conn quic.Connection) {
	str, err := conn.OpenStream()
	if err != nil {
		return
	}
	defer func(str quic.Stream) {
		_ = str.Close()
		_ = conn.CloseWithError(200, "sending a ping message expecting a pong reply, but the reply is not a pong message.")
	}(str)
	msg := NewPingMessage(int64(str.StreamID()), nil)
	_, err = msg.WriteTo(str)
	if err != nil {
		// TODO
		return
	}
	parser, err := NewServerMessageParser(str)
	if err != nil {
		// TODO
		return
	}
	if parser.Type != Pong {
		// TODO
		return
	}
	buf := make([]byte, 1024)
	for {
		n, err := str.Read(buf)
		if err != nil {
			// TODO logger print
			return
		}
		// TODO logger print
		fmt.Printf("received ping message: (%s) from client \n", string(buf[:n]))
		_, err = str.Write([]byte("ping"))
		if err != nil {
			// TODO logger print
			return
		}
		time.Sleep(time.Second * 5)
	}
}
