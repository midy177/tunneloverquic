package tunneloverquic

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/quic-go/quic-go"
	"log"
	"math/big"
	"net"
)

var (
	errFailedAuth = errors.New("failed authentication")
	errFirstConn  = errors.New("the first connection must be used for authorization verification")
)

type Hijacker func() (net.Conn, *bufio.ReadWriter, error)
type Authorizer func(msg []byte) (clientKey string, authed bool, err error)
type TLSConfigurator func() *tls.Config
type QuicConfigurator func() *quic.Config

func defaultAuth(msg []byte) (clientKey string, authed bool, err error) {
	return "", true, err
}

// Set up a bare-bones QUIC config for the Server
func defaultQuicConfig() *quic.Config {
	return &quic.Config{}
}

// Set up a bare-bones TLS config for the Server
func defaultTLSConfig() *tls.Config {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	// 将私钥编码为 PEM 格式
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}
}

type Server struct {
	serverAddr string
	Sessions   SessionManager
	authorizer Authorizer
	tlsConfig  TLSConfigurator
	quicConfig QuicConfigurator
}

func NewServer(addr string) *Server {
	return &Server{
		serverAddr: addr,
		tlsConfig:  defaultTLSConfig,
		quicConfig: defaultQuicConfig,
	}
}
func (l *Server) SetAuthorizer(auth Authorizer) *Server {
	l.authorizer = auth
	return l
}

func (l *Server) SetTlsConfig(tlsCfg TLSConfigurator) *Server {
	l.tlsConfig = tlsCfg
	return l
}

func (l *Server) SetQuicConfig(quicCfg QuicConfigurator) *Server {
	l.quicConfig = quicCfg
	return l
}

func (l *Server) Run() {
	ln, err := quic.ListenAddr(l.serverAddr, l.tlsConfig(), &quic.Config{})
	if err != nil {
		panic(err)
	}
	for {
		conn, err := ln.Accept(context.Background())
		if err != nil {
			log.Print(err.Error())
			continue
		}
		go l.handle(conn)
	}
}

func (l *Server) handle(conn quic.Connection) {
	streamHandle := &StreamHandle{
		server: l,
		conn:   conn,
	}
	streamHandle.connHandle()
}

type StreamHandle struct {
	server    *Server
	clientKey string
	conn      quic.Connection
}

func (l *StreamHandle) connHandle() {
	first := true
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		str, err := l.conn.AcceptStream(ctx) // for bidirectional streams
		var nerr net.Error
		if errors.As(err, &nerr) && nerr.Timeout() {
			// TODO  日志答应
			return
		}
		if first {
			err := l.streamAuth(str)
			if err != nil {
				// TODO logger print
				return
			}
			first = false
		} else {
			go l.streamHandle(str)
		}
	}
}

func (l *StreamHandle) streamAuth(str quic.Stream) error {
	defer str.Close()
	msg, err := NewServerMessageParser(str)
	if err != nil {
		return err
	}
	if msg.Type != Auth {
		return errFirstConn
	}
	clientKey, authed, err := l.server.authorizer(msg.Body())
	if err != nil {
		return err
	}
	if !authed {
		return errFailedAuth
	}
	if clientKey == "" {
		clientKey = l.conn.RemoteAddr().String()
	}
	// TODO add to clients slice
	l.clientKey = clientKey
	fmt.Println(clientKey)

	return nil
}

func (l *StreamHandle) streamHandle(str quic.Stream) {
	defer str.Close()
	msg, err := NewServerMessageParser(str)
	if err != nil {
		return
	}
	switch msg.Type {
	case Auth:

	case Connect:

	case Other:

	case Ping:
		l.keepAlive(str)
	case Pong:

	case Error:

	}
}

func (l *StreamHandle) keepAlive(str quic.Stream) {
	msg := NewPongMessage(int64(str.StreamID()), nil)
	_, err := msg.WriteTo(str)
	if err != nil {
		return
	}
	buf := make([]byte, 1024)
	for {
		n, err := str.Read(buf)
		if err != nil {
			// TODO logger print
			return
		}
		// TODO logger print
		fmt.Printf("received ping message: (%s) from client %s\n", string(buf[:n]), l.clientKey)
		_, err = str.Write([]byte("pong"))
		if err != nil {
			// TODO logger print
			return
		}
	}
}
