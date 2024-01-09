package main

import (
	"fmt"
	"tunneloverquic"
)

func main() {
	toq := tunneloverquic.NewServer("0.0.0.0:3000")
	toq.SetAuthorizer(func(msg []byte) (clientKey string, authed bool, err error) {
		fmt.Println(string(msg))
		return string(msg), true, nil
	})
	toq.Run()
}
