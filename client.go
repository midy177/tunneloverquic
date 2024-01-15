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
			return &quic.Config{
				KeepAlivePeriod: time.Second * 5,
			}
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
