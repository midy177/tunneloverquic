package main

import (
	"fmt"
	"net/http"
	"time"
	"tunneloverquic"
)

func main() {
	toq := tunneloverquic.NewServer("0.0.0.0:3001")
	toq.SetAuthorizer(func(msg []byte) (clientKey string, authed bool, err error) {
		fmt.Println(string(msg))
		return string(msg), true, nil
	})
	go toq.Run()

	for {
		time.Sleep(time.Second * 3)
		dialer, err := toq.GetDialer("adadjksjhfdfgh")
		if err != nil {
			continue
		}
		doHttpClient(dialer)
	}
}

func doHttpClient(dialer tunneloverquic.Dialer) {
	// 使用自定义的 Transport 创建 HTTP 客户端
	client := &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			DialContext: dialer,
			// 更多 Transport 的配置...
		}, // 更多客户端的配置...
		// 更多客户端的配置...
	}
	res, err := client.Get("https://gitlab.yeastar.com")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s\n", res.Status)
}
