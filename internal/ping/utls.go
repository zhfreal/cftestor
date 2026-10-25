package ping

import (
	"bufio"
	"context"
	"fmt"
	"io"
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

type utlsBodyCloser struct {
	io.ReadCloser
	tr   *UTLSTransport
	once sync.Once
}

func (c *utlsBodyCloser) Close() error {
	var err error
	c.once.Do(func() {
		if c.ReadCloser != nil {
			err = c.ReadCloser.Close()
		}
		c.tr.CloseIdleConnections()
	})
	return err
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
		tr1 := &http.Transport{
			DialContext:       outbound.GetDialContextByAddr(targetAddr),
			DisableKeepAlives: true,
		}
		b.mu.Lock()
		b.tr1 = tr1
		b.mu.Unlock()
		resp, err := tr1.RoundTrip(req)
		b.tlsHandShookAt = time.Now()
		b.responseAt = time.Now()
		if resp != nil && resp.Body != nil {
			resp.Body = &utlsBodyCloser{ReadCloser: resp.Body, tr: b}
		}
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
	conn, err := outbound.OutboundDialContext(req.Context(), "tcp", b.hostWithPort)
	if err != nil {
		return nil, fmt.Errorf("tcp net dial fail: %w", err)
	}
	b.mu.Lock()
	b.conn = conn
	b.mu.Unlock()

	tlsConn, err := b.tlsConnect(conn, req)
	if err != nil {
		b.CloseIdleConnections()
		return nil, fmt.Errorf("tls connect fail: %w", err)
	}
	b.mu.Lock()
	b.tlsConn = tlsConn
	b.mu.Unlock()

	b.tlsHandShookAt = time.Now()
	httpVersion := tlsConn.ConnectionState().NegotiatedProtocol
	resp := &http.Response{}
	switch httpVersion {
	case "h2":
		var h2_conn *http2.ClientConn
		h2_conn, err = b.tr2.NewClientConn(tlsConn)
		if err != nil {
			b.CloseIdleConnections()
			resp, err = nil, fmt.Errorf("create http2 client with connection fail: %w", err)
		} else {
			b.mu.Lock()
			b.h2Conn = h2_conn
			b.mu.Unlock()
			resp, err = h2_conn.RoundTrip(req)
			if err != nil {
				b.CloseIdleConnections()
			}
		}
	case "http/1.1", "":
		err = req.Write(tlsConn)
		if err != nil {
			b.CloseIdleConnections()
			resp, err = nil, fmt.Errorf("write http1 tls connection fail: %w", err)
		} else {
			resp, err = http.ReadResponse(bufio.NewReader(tlsConn), req)
			if err != nil {
				b.CloseIdleConnections()
			}
		}
	default:
		b.CloseIdleConnections()
		resp, err = nil, fmt.Errorf("unsupported http version: %s", httpVersion)
	}
	b.responseAt = time.Now()
	if resp != nil && resp.Body != nil {
		resp.Body = &utlsBodyCloser{ReadCloser: resp.Body, tr: b}
	}
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
	if b.h2Conn != nil {
		_ = b.h2Conn.Close()
		b.h2Conn = nil
	}
	if b.tlsConn != nil {
		_ = b.tlsConn.Close()
		b.tlsConn = nil
	}
	if b.conn != nil {
		_ = b.conn.Close()
		b.conn = nil
	}
	if b.tr1 != nil {
		b.tr1.CloseIdleConnections()
		b.tr1 = nil
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
