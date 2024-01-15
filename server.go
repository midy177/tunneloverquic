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
	clients    *clientsManager
	authorizer Authorizer
	tlsConfig  TLSConfigurator
	quicConfig QuicConfigurator
}

func NewServer(addr string) *Server {
	return &Server{
		serverAddr: addr,
		hijacker: func(msg *Message, str quic.Stream) (next bool) {
			return true
		},
		clients:    newClientsManager(),
		tlsConfig:  defaultTLSConfig,
		quicConfig: defaultQuicConfig,
	}
}

func (s *Server) SetHijacker(hijacker Hijacker) *Server {
	s.hijacker = hijacker
	return s
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
		clients:  s.clients,
		hijacker: s.hijacker,
		conn:     conn,
	}
	streamHandle.connHandle(true, s.authorizer)
}

func (s *Server) GetDialer(clientKey string) (Dialer, error) {
	return s.clients.getDialer(clientKey)
}

type ConnectHandle struct {
	clientKey string
	clients   *clientsManager
	hijacker  Hijacker
	conn      quic.Connection
}

func (l *ConnectHandle) connHandle(first bool, auth Authorizer) {
	firstConn := first
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer l.clients.Remove(l.clientKey)
	l.conn.ConnectionState()
	for {
		str, err := l.conn.AcceptStream(ctx) // for bidirectional streams
		if err != nil {
			// TODO logger print
			return
		}
		// Execute as server runtime
		if firstConn {
			err := l.streamAuth(str, auth)
			if err != nil {
				_ = l.conn.CloseWithError(200, "server authorization failed")
				// TODO logger print
				return
			}
			firstConn = false
		} else {
			go l.streamHandle(str)
		}
	}
}

func (l *ConnectHandle) streamAuth(str quic.Stream, auth Authorizer) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()
	defer str.Close()
	msg, err := newServerMessageParser(str)
	if err != nil {
		return err
	}
	if msg.MessageType != Auth {
		return errFirstConn
	}
	clientKey, authed, err := auth(msg.Body())
	if err != nil {
		_, _ = str.Write([]byte("failed"))
		return err
	}
	if !authed {
		_, _ = str.Write([]byte("failed"))
		return errFailedAuth
	}
	_, err = str.Write([]byte("ok"))
	if err != nil {
		return err
	}
	if clientKey == "" {
		clientKey = l.conn.RemoteAddr().String()
	}
	// TODO add to clients clients list
	l.clients.Add(clientKey, l.conn)
	l.clientKey = clientKey
	return nil
}

func (l *ConnectHandle) streamHandle(str quic.Stream) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()
	defer str.Close()
	msg, err := newServerMessageParser(str)
	if err != nil {
		return
	}
	if !l.hijacker(msg, str) {
		return
	}
	switch msg.MessageType {
	case Auth:

	case Connect:
		fmt.Printf("clientKey %s -> proto: %s address: %s\n", l.clientKey, msg.Proto, msg.Address)
		err := clientDial(str, msg)
		if err != nil {
			// TODO
			return
		}
	case Other:

	case Pong:

	case Error:
	default:
		return

	}
}

func (l *ConnectHandle) SetHijacker(hijacker Hijacker) *ConnectHandle {
	l.hijacker = hijacker
	return l
}

func (l *ConnectHandle) GetDialer() (Dialer, error) {
	str, err := l.conn.OpenStreamSync(context.TODO())
	if err != nil {
		l.clients.Remove(l.clientKey)
		return nil, err
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		return newConnection(str, network, address)
	}, nil
}
