package tunneloverquic

import (
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

type Hijacker func(msg *Message, str quic.Stream) (next bool)
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
	hijacker   Hijacker
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
func (s *Server) SetAuthorizer(auth Authorizer) *Server {
	s.authorizer = auth
	return s
}

func (s *Server) SetTlsConfig(tlsCfg TLSConfigurator) *Server {
	s.tlsConfig = tlsCfg
	return s
}

func (s *Server) SetQuicConfig(quicCfg QuicConfigurator) *Server {
	s.quicConfig = quicCfg
	return s
}

func (s *Server) Run() {
	ln, err := quic.ListenAddr(s.serverAddr, s.tlsConfig(), &quic.Config{})
	if err != nil {
		panic(err)
	}
	println("quic listen on: " + s.serverAddr)
	for {
		conn, err := ln.Accept(context.Background())
		if err != nil {
			log.Print(err.Error())
			continue
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn quic.Connection) {
	streamHandle := &ConnectHandle{
		hijacker: s.hijacker,
		Conn:     conn,
	}
	streamHandle.ConnHandle(true, s.authorizer)
}

type ConnectHandle struct {
	clientKey string
	hijacker  Hijacker
	Conn      quic.Connection
}

func (l *ConnectHandle) ConnHandle(first bool, auth Authorizer) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		str, err := l.Conn.AcceptStream(ctx) // for bidirectional streams
		var nerr net.Error
		if errors.As(err, &nerr) && nerr.Timeout() {
			// TODO logger print
			return
		}
		// Execute as server runtime
		if first {
			err := l.streamAuth(str, auth)
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

func (l *ConnectHandle) streamAuth(str quic.Stream, auth Authorizer) error {
	defer str.Close()
	msg, err := NewServerMessageParser(str)
	if err != nil {
		return err
	}
	if msg.Type != Auth {
		return errFirstConn
	}
	clientKey, authed, err := auth(msg.Body())
	if err != nil {
		return err
	}
	if !authed {
		return errFailedAuth
	}
	if clientKey == "" {
		clientKey = l.Conn.RemoteAddr().String()
	}
	// TODO add to clients session list
	l.clientKey = clientKey
	fmt.Println(clientKey)

	return nil
}

func (l *ConnectHandle) streamHandle(str quic.Stream) {
	defer str.Close()
	msg, err := NewServerMessageParser(str)
	if err != nil {
		return
	}
	if !l.hijacker(msg, str) {
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

func (l *ConnectHandle) keepAlive(str quic.Stream) {
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

func (l *ConnectHandle) SetHijacker(hijacker Hijacker) *ConnectHandle {
	l.hijacker = hijacker
	return l
}

func (l *ConnectHandle) GetDialer() Dialer {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, nil
	}
}
