package tunneloverquic

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/quic-go/quic-go"
	"io"
	"strings"
)

type messageType int64

const (
	Auth messageType = iota + 1
	Connect
	Other
	Ping
	Pong
	Error
)

type Message struct {
	connID      int64
	MessageType messageType
	bytes       []byte
	Proto       string
	Address     string
}

func newAuthMessage(connID int64, data []byte) *Message {
	return &Message{
		connID:      connID,
		MessageType: Auth,
		bytes:       data,
	}
}

func newConnectMessage(connID int64, proto, address string) *Message {
	return &Message{
		connID:      connID,
		MessageType: Connect,
		bytes:       []byte(fmt.Sprintf("%s/%s", proto, address)),
		Proto:       proto,
		Address:     address,
	}
}

func newPingMessage(connID int64, data []byte) *Message {
	return &Message{
		connID:      connID,
		MessageType: Ping,
		bytes:       data,
	}
}

func newPongMessage(connID int64, data []byte) *Message {
	return &Message{
		connID:      connID,
		MessageType: Pong,
		bytes:       data,
	}
}

func newServerMessageParser(reader io.Reader) (*Message, error) {
	buf := bufio.NewReader(reader)

	connID, err := binary.ReadVarint(buf)
	if err != nil {
		return nil, err
	}

	mType, err := binary.ReadVarint(buf)
	if err != nil {
		return nil, err
	}

	m := &Message{
		MessageType: messageType(mType),
		connID:      connID,
	}
	// 获取数据部分缓冲区大小
	space, err := binary.ReadVarint(buf)
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(io.LimitReader(buf, space))
	if err != nil {
		return nil, err
	}
	m.bytes = bytes
	if m.MessageType == Connect {
		parts := strings.SplitN(string(bytes), "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("failed to parse connect Address")
		}
		m.Proto = parts[0]
		m.Address = parts[1]
	}

	return m, nil
}

func (m *Message) Body() []byte {
	return m.bytes
}

func (m *Message) Bytes() []byte {
	bytesLength := len(m.bytes)
	space := bytesLength + 24
	// Calculate required buffer size
	buf := make([]byte, space)
	// offset of header data
	offset := m.header(buf)
	// Copy message data to buffer
	copy(buf[offset:], m.bytes)
	return buf[:offset+bytesLength]
}

func (m *Message) header(buf []byte) int {
	// 头部数据的偏移量
	offset := 0

	// 将各种头部信息写入缓冲区
	offset += binary.PutVarint(buf[offset:], m.connID)
	offset += binary.PutVarint(buf[offset:], int64(m.MessageType))
	offset += binary.PutVarint(buf[offset:], int64(len(m.bytes)))

	return offset
}

func (m *Message) WriteTo(str quic.Stream) (int, error) {
	n, err := str.Write(m.Bytes())
	if err != nil {
		return n, err
	}
	return len(m.bytes), err
}

func (m *Message) ToString() string {
	switch m.MessageType {
	case Auth:
		return fmt.Sprintf("AUTH    [%d]: %s", m.connID, string(m.bytes))
	case Connect:
		return fmt.Sprintf("CONNECT [%d]: %s/%s", m.connID, m.Proto, m.Address)
	case Other:
		return fmt.Sprintf("OTHER   [%d]: %s", m.connID, string(m.bytes))
	default:
		return fmt.Sprintf("UNKNOWN [%d]: %d", m.connID, m.MessageType)
	}
}

func (m *Message) Network() string {
	return m.Proto
}

func (m *Message) String() string {
	return m.Address
}
