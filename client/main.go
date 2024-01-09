package main

import (
	"fmt"
	"tunneloverquic"
)

func main() {
	_, err := tunneloverquic.ClientConnect("192.168.12.40:3001", []byte("adadjksjhfdfgh"), nil, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	//dialer := connect.GetDialer()
	//// 使用自定义的 Dialer 创建 Transport
	//transport := &http.Transport{
	//	DialContext: dialer,
	//	// 更多 Transport 的配置...
	//}
	//
	//// 使用自定义的 Transport 创建 HTTP 客户端
	//client := &http.Client{
	//	Transport: transport,
	//	// 更多客户端的配置...
	//}
	//res, err := client.Get("http://cert-system.yeastar.com/")
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//fmt.Printf("%s\n", res.Status)
	select {}
}
