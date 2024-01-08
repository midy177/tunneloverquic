package main

import "tunneloverquic"

func main() {
	toq := tunneloverquic.NewServer("0.0.0.0:3000")
	toq.Run()
}
