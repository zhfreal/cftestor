package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

func NewUTLSTransport(helloID utls.ClientHelloID, hostWithPort string, timeout time.Duration) *UTLSTransport {
	return &UTLSTransport{clientHello: helloID, hostWithPort: hostWithPort, timeout: timeout}
}

type UTLSTransport struct {
	tr1 http.Transport
	tr2 http2.Transport

	mu           sync.RWMutex
	clientHello  utls.ClientHelloID
	hostWithPort string
	startAt      time.Time
	tlsShakedAt  time.Time
	responseAt   time.Time
	timeout      time.Duration
	conn         net.Conn
	h2Conn       *http2.ClientConn
	tlsConn      *utls.UConn
}

func (b *UTLSTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Scheme {
	case "https":
		return b.httpsRoundTrip(req)
	case "http":
		return b.tr1.RoundTrip(req)
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", req.URL.Scheme)
	}
}

func (b *UTLSTransport) httpsRoundTrip(req *http.Request) (*http.Response, error) {
	if len(b.hostWithPort) == 0 {
		port := req.URL.Port()
		if port == "" {
			port = "443"
		}
		b.hostWithPort = fmt.Sprintf("%s:%s", req.URL.Host, port)
	}

	b.startAt = time.Now()
	var err error
	b.conn, err = net.DialTimeout("tcp", b.hostWithPort, 600*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("tcp net dial fail: %w", err)
	}
	// defer conn.Close() // nolint

	b.tlsConn, err = b.tlsConnect(b.conn, req)
	b.tlsShakedAt = time.Now()
	if err != nil {
		return nil, fmt.Errorf("tls connect fail: %w", err)
	}
	b.tlsShakedAt = time.Now()
	httpVersion := b.tlsConn.ConnectionState().NegotiatedProtocol
	resp := &http.Response{}
	switch httpVersion {
	case "h2":
		var h2_conn *http2.ClientConn
		h2_conn, err = b.tr2.NewClientConn(b.tlsConn)
		b.h2Conn = h2_conn
		if err != nil {
			resp, err = nil, fmt.Errorf("create http2 client with connection fail: %w", err)
		} else {
			// defer h2_conn.Close() // nolint
			resp, err = h2_conn.RoundTrip(req)
		}
	case "http/1.1", "":
		err = req.Write(b.tlsConn)
		if err != nil {
			resp, err = nil, fmt.Errorf("write http1 tls connection fail: %w", err)
		} else {
			resp, err = http.ReadResponse(bufio.NewReader(b.tlsConn), req)
		}
	default:
		resp, err = nil, fmt.Errorf("unsuported http version: %s", httpVersion)
	}
	b.responseAt = time.Now()
	return resp, err
}

func (b *UTLSTransport) getTLSConfig(req *http.Request) *utls.Config {
	host_name, _, err := net.SplitHostPort(req.URL.Host)
	if err != nil {
		host_name = req.URL.Host
	}
	return &utls.Config{
		ServerName:         host_name,
		InsecureSkipVerify: false,
	}
}

func (b *UTLSTransport) tlsConnect(conn net.Conn, req *http.Request) (*utls.UConn, error) {
	b.mu.RLock()
	tlsConn := utls.UClient(conn, b.getTLSConfig(req), b.clientHello)
	b.mu.RUnlock()

	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("tls handshake fail: %w", err)
	}
	return tlsConn, nil
}

func (b *UTLSTransport) Stat() (time.Duration, time.Duration) {
	return b.tlsShakedAt.Sub(b.startAt), b.responseAt.Sub(b.tlsShakedAt)
}

func (b *UTLSTransport) SetClientHello(hello utls.ClientHelloID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clientHello = hello
}

func (b *UTLSTransport) CloseIdleConnections() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn != nil {
		b.conn.Close()
	}
	if b.tlsConn != nil {
		b.tlsConn.Close()
	}
	if b.h2Conn != nil {
		b.h2Conn.Close()
	}
	b.tr1.CloseIdleConnections()
}

func newHttpClient(helloID utls.ClientHelloID, hostWithPort string, timeout time.Duration) (*http.Client, *UTLSTransport) {
	tr := NewUTLSTransport(helloID, hostWithPort, timeout)
	var client = &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}
	return client, tr
}

func performUtlsDial(host string, hostName string, timeout time.Duration, hellID utls.ClientHelloID) bool {
	dialer := net.Dialer{Timeout: timeout}
	dialConn, err := dialer.Dial("tcp", host)
	if err != nil {
		return false
	}
	conf := &utls.Config{
		ServerName: hostName,
	}
	tlsConn := utls.UClient(dialConn, conf, hellID)
	err = tlsConn.Handshake()
	_ = dialConn.Close()
	return err == nil
}
