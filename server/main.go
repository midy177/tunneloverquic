package main

import (
	"fmt"
	"net/http"
	"time"
	"tunneloverquic"
)

func main() {
	toq := tunneloverquic.NewServer("0.0.0.0:3000")
	toq.SetAuthorizer(func(msg []byte) (clientKey string, authed bool, err error) {
		fmt.Println(string(msg))
		return string(msg), true, nil
	})
	go func() {
	setup:
		dialer, err := toq.GetDialer("adadjksjhfdfgh")
		if err != nil {
			time.Sleep(time.Second * 3)
			goto setup
		}
		// 使用自定义的 Transport 创建 HTTP 客户端
		client := &http.Client{
			Transport: &http.Transport{
				DialContext: dialer,
				// 更多 Transport 的配置...
			}, // 更多客户端的配置...
		}
		client.Timeout = time.Second * 5
		for {
			time.Sleep(time.Second * 3)
			res, err := client.Get("https://gitlab.yeastar.com/users/sign_in")
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Printf("%s\n", res.Status)
		}
	}()
	toq.Run()
}
