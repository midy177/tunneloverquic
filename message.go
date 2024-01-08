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
	connID  int64
	Type    messageType
	bytes   []byte
	Proto   string
	Address string
}

func NewAuthMessage(connID int64, data []byte) *Message {
	return &Message{
		connID: connID,
		Type:   Auth,
		bytes:  data,
	}
}

func NewConnectMessage(connID int64, proto, address string) *Message {
	return &Message{
		connID:  connID,
		Type:    Connect,
		bytes:   []byte(fmt.Sprintf("%s/%s", proto, address)),
		Proto:   proto,
		Address: address,
	}
}

func NewPingMessage(connID int64, data []byte) *Message {
	return &Message{
		connID: connID,
		Type:   Ping,
		bytes:  data,
	}
}

func NewPongMessage(connID int64, data []byte) *Message {
	return &Message{
		connID: connID,
		Type:   Pong,
		bytes:  data,
	}
}

func NewServerMessageParser(reader io.Reader) (*Message, error) {
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
		Type:   messageType(mType),
		connID: connID,
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
	if m.Type == Connect {
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
	// 计算所需缓冲区大小
	space := len(m.bytes)
	buf := make([]byte, 24+space)
	// 头部数据的偏移量
	offset := m.header(buf)
	// 将消息数据拷贝到缓冲区
	copy(buf[offset:], m.bytes)
	return buf
}

func (m *Message) header(buf []byte) int {
	// 头部数据的偏移量
	offset := 0

	// 将各种头部信息写入缓冲区
	offset += binary.PutVarint(buf[offset:], m.connID)
	offset += binary.PutVarint(buf[offset:], int64(m.Type))
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

func (m *Message) String() string {
	switch m.Type {
	case Auth:
		return fmt.Sprintf("AUTH    [%d]: %s", m.connID, string(m.bytes))
	case Connect:
		return fmt.Sprintf("CONNECT [%d]: %s/%s", m.connID, m.Proto, m.Address)
	case Other:
		return fmt.Sprintf("OTHER   [%d]: %s", m.connID, string(m.bytes))
	default:
		return fmt.Sprintf("UNKNOWN [%d]: %d", m.connID, m.Type)
	}
}