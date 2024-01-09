package tunneloverquic

import (
	"github.com/quic-go/quic-go"
	"net"
	"time"
)

type connection struct {
	stream  quic.Stream
	message *Message
}

func newConnection(str quic.Stream, network, address string) (*connection, error) {
	// open connect
	msg := newConnectMessage(int64(str.StreamID()), network, address)
	_, err := msg.WriteTo(str)
	if err != nil {
		return nil, err
	}
	return &connection{
		stream:  str,
		message: msg,
	}, nil
}
func (c *connection) ConnID() int64 {
	return int64(c.stream.StreamID())
}
func (c *connection) Read(b []byte) (n int, err error) {
	return c.stream.Read(b)
}

func (c *connection) Write(b []byte) (n int, err error) {
	return c.stream.Write(b)
}

func (c *connection) Close() error {
	return c.stream.Close()
}

func (c *connection) LocalAddr() net.Addr {
	return c.message
}

func (c *connection) RemoteAddr() net.Addr {
	return c.message
}

func (c *connection) SetDeadline(t time.Time) error {
	return c.stream.SetDeadline(t)
}

func (c *connection) SetReadDeadline(t time.Time) error {
	return c.stream.SetReadDeadline(t)
}

func (c *connection) SetWriteDeadline(t time.Time) error {
	return c.stream.SetWriteDeadline(t)
}
