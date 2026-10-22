package ping

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"cftestor/internal/outbound"
	utls "github.com/metacubex/utls"
	"golang.org/x/net/http2"
)

func NewUTLSTransport(helloID utls.ClientHelloID, hostWithPort string, timeout time.Duration) *UTLSTransport {
	return &UTLSTransport{clientHello: helloID, hostWithPort: hostWithPort, timeout: timeout}
}

type UTLSTransport struct {
	tr1 *http.Transport
	tr2 *http2.Transport

	mu             sync.RWMutex
	clientHello    utls.ClientHelloID
	hostWithPort   string
	startAt        time.Time
	timeout        time.Duration
	tlsHandShookAt time.Time
	responseAt     time.Time
	conn           net.Conn
	h2Conn         *http2.ClientConn
	tlsConn        *utls.UConn
}

func (b *UTLSTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Scheme {
	case "https":
		b.tr2 = &http2.Transport{}
		return b.httpsRoundTrip(req)
	case "http":
		b.startAt = time.Now()
		targetAddr := b.hostWithPort
		if len(targetAddr) == 0 {
			targetAddr = req.URL.Host
			if !strings.Contains(targetAddr, ":") {
				targetAddr = net.JoinHostPort(targetAddr, "80")
			}
		}
		b.tr1 = &http.Transport{
			DialContext:       outbound.GetDialContextByAddr(targetAddr),
			DisableKeepAlives: true,
		}
		resp, err := b.tr1.RoundTrip(req)
		b.tlsHandShookAt = time.Now()
		b.responseAt = time.Now()
		return resp, err
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
		b.hostWithPort = net.JoinHostPort(req.URL.Hostname(), port)
	}

	b.startAt = time.Now()
	var err error
	b.conn, err = outbound.OutboundDialContext(req.Context(), "tcp", b.hostWithPort)
	if err != nil {
		return nil, fmt.Errorf("tcp net dial fail: %w", err)
	}

	b.tlsConn, err = b.tlsConnect(b.conn, req)
	if err != nil {
		_ = b.conn.Close()
		return nil, fmt.Errorf("tls connect fail: %w", err)
	}
	b.tlsHandShookAt = time.Now()
	httpVersion := b.tlsConn.ConnectionState().NegotiatedProtocol
	resp := &http.Response{}
	switch httpVersion {
	case "h2":
		var h2_conn *http2.ClientConn
		h2_conn, err = b.tr2.NewClientConn(b.tlsConn)
		b.h2Conn = h2_conn
		if err != nil {
			_ = b.tlsConn.Close()
			_ = b.conn.Close()
			resp, err = nil, fmt.Errorf("create http2 client with connection fail: %w", err)
		} else {
			resp, err = h2_conn.RoundTrip(req)
		}
	case "http/1.1", "":
		err = req.Write(b.tlsConn)
		if err != nil {
			_ = b.tlsConn.Close()
			_ = b.conn.Close()
			resp, err = nil, fmt.Errorf("write http1 tls connection fail: %w", err)
		} else {
			resp, err = http.ReadResponse(bufio.NewReader(b.tlsConn), req)
			if err != nil {
				_ = b.tlsConn.Close()
				_ = b.conn.Close()
			}
		}
	default:
		_ = b.tlsConn.Close()
		_ = b.conn.Close()
		resp, err = nil, fmt.Errorf("unsupported http version: %s", httpVersion)
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

	if err := tlsConn.HandshakeContext(req.Context()); err != nil {
		return nil, fmt.Errorf("tls handshake fail: %w", err)
	}
	return tlsConn, nil
}

func (b *UTLSTransport) Stat() (time.Duration, time.Duration) {
	return b.tlsHandShookAt.Sub(b.startAt), b.responseAt.Sub(b.tlsHandShookAt)
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
	if b.tr1 != nil {
		b.tr1.CloseIdleConnections()
	}
}

func NewHttpClient(helloID utls.ClientHelloID, hostWithPort string, timeout time.Duration) (*http.Client, *UTLSTransport) {
	tr := NewUTLSTransport(helloID, hostWithPort, timeout)
	var client = &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}
	return client, tr
}

func PerformUtlsDial(host string, hostNameStr string, timeout time.Duration, hellID utls.ClientHelloID) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dialConn, err := outbound.OutboundDialContext(ctx, "tcp", host)
	if err != nil {
		return false
	}
	defer dialConn.Close()

	conf := &utls.Config{
		ServerName: hostNameStr,
	}
	tlsConn := utls.UClient(dialConn, conf, hellID)
	defer tlsConn.Close()

	err = tlsConn.HandshakeContext(ctx)
	return err == nil
}
