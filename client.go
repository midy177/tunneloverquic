package tunneloverquic

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"time"
)

func ClientConnect(addr, auth string, tlsConfig TLSConfigurator, quicConfig QuicConfigurator) (*ConnectHandle, error) {
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
	conn, err := quic.DialAddr(ctx, addr, tlsConfig(), nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	streamHandle := &ConnectHandle{
		Conn: conn,
	}
	go streamHandle.connHandle(func(msg []byte) (clientKey string, authed bool, err error) {
		return "local", true, err
	})
	return streamHandle, nil
}
