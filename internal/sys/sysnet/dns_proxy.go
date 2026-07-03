package sysnet

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"golang.org/x/net/proxy"
)

const dohURL = "https://8.8.8.8/dns-query"

// DNSProxy is a lightweight UDP DNS proxy that forwards queries to a DoH server
// via a SOCKS5 proxy, preventing DNS leaks in non-TUN proxy modes.
//
// Routing: system DNS → DNSProxy (127.0.0.1:53) → DoH via SOCKS5 (mixed inbound)
// → sing-box proxy outbound → remote VPN server → Google DNS.
//
// Requires root on macOS to bind port 53.
type DNSProxy struct {
	// ListenAddr is the address to listen on, e.g. "127.0.0.1:53".
	ListenAddr string
	// SocksAddr is the SOCKS5 address to forward through, e.g. "127.0.0.1:7890".
	// If empty, DoH requests are made directly without a proxy.
	SocksAddr string

	conn    *net.UDPConn
	client  *http.Client
	stopped int32
	done    chan struct{}
}

// NewDNSProxy creates a proxy that listens on listenAddr and forwards
// DNS-over-HTTPS through the SOCKS5 endpoint at socksAddr.
func NewDNSProxy(listenAddr, socksAddr string) *DNSProxy {
	return &DNSProxy{
		ListenAddr: listenAddr,
		SocksAddr:  socksAddr,
		done:       make(chan struct{}),
	}
}

// Start binds the UDP listener and begins serving DNS queries.
func (p *DNSProxy) Start() error {
	addr, err := net.ResolveUDPAddr("udp4", p.ListenAddr)
	if err != nil {
		return fmt.Errorf("dns proxy: resolve %s: %w", p.ListenAddr, err)
	}
	p.conn, err = net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("dns proxy: listen %s: %w", p.ListenAddr, err)
	}
	p.client = p.buildClient()
	go p.serve()
	return nil
}

// Stop shuts down the listener and waits for the serve goroutine to exit.
func (p *DNSProxy) Stop() {
	if atomic.CompareAndSwapInt32(&p.stopped, 0, 1) {
		if p.conn != nil {
			p.conn.Close()
		}
		<-p.done
	}
}

func (p *DNSProxy) serve() {
	defer close(p.done)
	buf := make([]byte, 4096)
	for {
		p.conn.SetReadDeadline(time.Now().Add(time.Second))
		n, src, err := p.conn.ReadFromUDP(buf)
		if err != nil {
			if atomic.LoadInt32(&p.stopped) == 1 {
				return
			}
			continue
		}
		query := make([]byte, n)
		copy(query, buf[:n])
		go p.handle(query, src)
	}
}

func (p *DNSProxy) handle(query []byte, src *net.UDPAddr) {
	resp, err := p.forwardDoH(query)
	if err != nil {
		return // Drop; the client will retry on timeout
	}
	_ = p.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, _ = p.conn.WriteToUDP(resp, src)
}

func (p *DNSProxy) forwardDoH(msg []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dohURL, bytes.NewReader(msg))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("doh status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (p *DNSProxy) buildClient() *http.Client {
	if p.SocksAddr != "" {
		d, err := proxy.SOCKS5("tcp", p.SocksAddr, nil, proxy.Direct)
		if err == nil {
			return &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						return d.Dial(network, addr)
					},
				},
				Timeout: 5 * time.Second,
			}
		}
	}
	return &http.Client{Timeout: 5 * time.Second}
}
