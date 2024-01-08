package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 3s handshake timeout
	defer cancel()
	conn, err := quic.DialAddr(ctx, "172.31.109.4:3000", &tls.Config{
		// 在生产环境中请配置适当的证书和密钥
		InsecureSkipVerify: true,
	}, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	str, err := conn.OpenStream()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func(str quic.Stream) {
		_ = str.Close()
	}(str)
	for i := 0; i < 1; i++ {
		_, err2 := str.Write([]byte("hello"))
		if err2 != nil {
			fmt.Println(err2.Error())
		}
		time.Sleep(time.Second)
	}
}
