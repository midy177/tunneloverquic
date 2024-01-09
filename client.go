package tunneloverquic

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/quic-go/quic-go"
	"io"
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
	if quicConfig == nil {
		quicConfig = func() *quic.Config {
			return nil
		}
	}
	conn, err := quic.DialAddr(ctx, addr, tlsConfig(), quicConfig())
	if err != nil {
		return nil, err
	}

	err = clientAuth(conn, auth)
	if err != nil {
		return nil, err
	}
	go clientPing(conn)

	streamHandle := &ConnectHandle{
		conn: conn,
	}
	streamHandle.SetHijacker(func(msg *Message, str quic.Stream) (next bool) {
		return true
	})
	go streamHandle.connHandle(false, func(msg []byte) (clientKey string, authed bool, err error) {
		return "local", true, err
	})
	return streamHandle, nil
}

func clientAuth(conn quic.Connection, auth []byte) error {
	str, err := conn.OpenStream()
	if err != nil {
		return err
	}
	defer func(str quic.Stream) {
		_ = str.Close()
	}(str)
	msg := newAuthMessage(int64(str.StreamID()), auth)
	_, err = msg.WriteTo(str)
	if err != nil {
		return err
	}
	time.Sleep(time.Second)
	buf := make([]byte, 1024)
	n, err := str.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}
	if !bytes.Equal(buf[:n], []byte("ok")) {
		return errors.New("server authorization failed")
	}
	fmt.Println("server authorization ok.")
	return nil
}

func clientPing(conn quic.Connection) {
	defer func(conn quic.Connection) {
		_ = conn.CloseWithError(200, "sending a ping Message expecting a pong reply, but the reply is not a pong Message.")
	}(conn)

	str, err := conn.OpenStream()
	if err != nil {
		return
	}
	defer func(str quic.Stream) {
		_ = str.Close()
	}(str)
	msg := newPingMessage(int64(str.StreamID()), nil)
	_, err = msg.WriteTo(str)
	if err != nil {
		// TODO
		return
	}
	buf := make([]byte, 64)
	for {
		n, err := str.Read(buf)
		if err != nil {
			// TODO logger print
			return
		}
		// TODO logger print
		fmt.Printf("pong message: %s\n", string(buf[:n]))
		time.Sleep(time.Second * 5)
		_, err = str.Write([]byte("ping"))
		if err != nil {
			// TODO logger print
			return
		}
	}
}
