package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func buildProxy(cfg Config, baseDialer *net.Dialer) (func(*http.Request) (*url.URL, error), func(context.Context, string, string) (net.Conn, error), error) {
	proxyFunc := http.ProxyFromEnvironment
	dialContext := baseDialer.DialContext

	proxyDialer := *baseDialer
	if cfg.ProxyDialTimeout > 0 {
		proxyDialer.Timeout = cfg.ProxyDialTimeout
	}

	if cfg.ProxyURL == "" {
		return proxyFunc, dialContext, nil
	}

	u, err := url.Parse(cfg.ProxyURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid proxy url %q: %w", cfg.ProxyURL, err)
	}

	username := cfg.ProxyUsername
	password := cfg.ProxyPassword
	if u.User != nil {
		username = u.User.Username()
		if p, ok := u.User.Password(); ok {
			password = p
		}
	}

	scheme := strings.ToLower(u.Scheme)

	switch {
	case scheme == "http" || scheme == "https":
		proxyFunc = http.ProxyURL(u)
		dialContext = proxyDialer.DialContext

	case strings.HasPrefix(scheme, "socks5"):
		proxyAddr := u.Host
		if !strings.Contains(proxyAddr, ":") {
			proxyAddr = net.JoinHostPort(proxyAddr, "1080")
		}

		dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialSOCKS5(ctx, &proxyDialer, proxyAddr, username, password, network, addr, cfg.ProxyHandshakeTimeout)
		}
		proxyFunc = nil

	case strings.HasPrefix(scheme, "socks4"):
		proxyAddr := u.Host
		if !strings.Contains(proxyAddr, ":") {
			proxyAddr = net.JoinHostPort(proxyAddr, "1080")
		}

		dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialSOCKS4(ctx, &proxyDialer, proxyAddr, username, network, addr, cfg.ProxyHandshakeTimeout)
		}
		proxyFunc = nil

	default:
		return nil, nil, fmt.Errorf("unsupported proxy scheme %q", u.Scheme)
	}

	return proxyFunc, dialContext, nil
}

func dialSOCKS5(ctx context.Context, dialer *net.Dialer, proxyAddr, username, password, network, addr string, handshakeTimeout time.Duration) (net.Conn, error) {
	conn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, err
	}

	if handshakeTimeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(handshakeTimeout))
	}

	if err := socks5Handshake(conn, username, password, addr); err != nil {
		conn.Close()
		return nil, err
	}

	if handshakeTimeout > 0 {
		_ = conn.SetDeadline(time.Time{})
	}

	return conn, nil
}

func socks5Handshake(conn net.Conn, username, password, destAddr string) error {
	const (
		version           = 0x05
		noAuth            = 0x00
		userPassAuth      = 0x02
		cmdConnect        = 0x01
		atypDomain        = 0x03
		authVersion       = 0x01
		replySucceeded    = 0x00
		authStatusSuccess = 0x00
	)

	methods := []byte{noAuth}
	if username != "" || password != "" {
		methods = []byte{noAuth, userPassAuth}
	}

	greet := []byte{version, byte(len(methods))}
	greet = append(greet, methods...)

	if _, err := conn.Write(greet); err != nil {
		return fmt.Errorf("socks5: write greeting: %w", err)
	}

	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return fmt.Errorf("socks5: read greeting response: %w", err)
	}
	if resp[0] != version {
		return fmt.Errorf("socks5: unexpected version %d", resp[0])
	}

	switch resp[1] {
	case noAuth:
		// no-op
	case userPassAuth:
		if len(username) > 255 || len(password) > 255 {
			return fmt.Errorf("socks5: username/password too long")
		}
		buf := make([]byte, 0, 3+len(username)+len(password))
		buf = append(buf, authVersion, byte(len(username)))
		buf = append(buf, []byte(username)...)
		buf = append(buf, byte(len(password)))
		buf = append(buf, []byte(password)...)

		if _, err := conn.Write(buf); err != nil {
			return fmt.Errorf("socks5: write auth request: %w", err)
		}

		authResp := make([]byte, 2)
		if _, err := io.ReadFull(conn, authResp); err != nil {
			return fmt.Errorf("socks5: read auth response: %w", err)
		}
		if authResp[1] != authStatusSuccess {
			return fmt.Errorf("socks5: auth failed (status=%d)", authResp[1])
		}
	default:
		return fmt.Errorf("socks5: unsupported auth method %d", resp[1])
	}

	host, portStr, err := net.SplitHostPort(destAddr)
	if err != nil {
		return fmt.Errorf("socks5: invalid target address %q: %w", destAddr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("socks5: invalid target port %q", portStr)
	}

	hostBytes := []byte(host)
	req := make([]byte, 0, 6+len(hostBytes))
	req = append(req, version, cmdConnect, 0x00 /* RSV */, atypDomain, byte(len(hostBytes)))
	req = append(req, hostBytes...)
	req = append(req, byte(port>>8), byte(port&0xff))

	if _, err := conn.Write(req); err != nil {
		return fmt.Errorf("socks5: write connect request: %w", err)
	}

	if _, err := io.ReadFull(conn, resp); err != nil {
		return fmt.Errorf("socks5: read connect response header: %w", err)
	}
	if resp[0] != version {
		return fmt.Errorf("socks5: unexpected response version %d", resp[0])
	}
	if resp[1] != replySucceeded {
		return fmt.Errorf("socks5: connect failed (reply=%d)", resp[1])
	}

	switch resp[3] {
	case 0x01:
		if err := discard(conn, 6); err != nil {
			return err
		}
	case 0x03:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return fmt.Errorf("socks5: read domain length: %w", err)
		}
		domainLen := int(lenBuf[0])
		if err := discard(conn, domainLen+2); err != nil {
			return err
		}
	case 0x04:
		if err := discard(conn, 18); err != nil {
			return err
		}
	default:
		return fmt.Errorf("socks5: unknown atyp %d", resp[3])
	}

	return nil
}

func dialSOCKS4(ctx context.Context, dialer *net.Dialer, proxyAddr, username, network, addr string, handshakeTimeout time.Duration) (net.Conn, error) {
	conn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, err
	}

	if handshakeTimeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(handshakeTimeout))
	}

	if err := socks4Handshake(conn, username, addr); err != nil {
		conn.Close()
		return nil, err
	}

	if handshakeTimeout > 0 {
		_ = conn.SetDeadline(time.Time{})
	}

	return conn, nil
}

func socks4Handshake(conn net.Conn, username, destAddr string) error {
	const (
		version        = 0x04
		cmdConnect     = 0x01
		replyGranted   = 0x5a
		socks4aFakeIP0 = 0x00
		socks4aFakeIP1 = 0x00
		socks4aFakeIP2 = 0x00
		socks4aFakeIP3 = 0x01
	)

	host, portStr, err := net.SplitHostPort(destAddr)
	if err != nil {
		return fmt.Errorf("socks4: invalid target address %q: %w", destAddr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("socks4: invalid target port %q", portStr)
	}

	ip := net.ParseIP(host).To4()
	useSocks4a := ip == nil

	buf := &bytes.Buffer{}
	buf.Grow(9 + len(username) + len(host))

	// VN, CD.
	buf.WriteByte(version)
	buf.WriteByte(cmdConnect)

	// DSTPORT.
	buf.WriteByte(byte(port >> 8))
	buf.WriteByte(byte(port & 0xff))

	// DSTIP.
	if useSocks4a {
		buf.Write([]byte{socks4aFakeIP0, socks4aFakeIP1, socks4aFakeIP2, socks4aFakeIP3})
	} else {
		buf.Write(ip)
	}

	// USERID (optional) + NUL.
	if username != "" {
		buf.WriteString(username)
	}
	buf.WriteByte(0x00)

	// For SOCKS4A, append target host + NUL.
	if useSocks4a {
		buf.WriteString(host)
		buf.WriteByte(0x00)
	}

	if _, err := conn.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("socks4: write connect request: %w", err)
	}

	resp := make([]byte, 8)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return fmt.Errorf("socks4: read connect response: %w", err)
	}

	if resp[1] != replyGranted {
		return fmt.Errorf("socks4: connect failed (reply=%d)", resp[1])
	}

	return nil
}

func discard(r io.Reader, n int) error {
	if n <= 0 {
		return nil
	}
	const chunk = 16
	buf := make([]byte, chunk)
	remaining := n
	for remaining > 0 {
		toRead := chunk
		if remaining < chunk {
			toRead = remaining
		}
		read, err := r.Read(buf[:toRead])
		if err != nil {
			return fmt.Errorf("discard: %w", err)
		}
		if read == 0 {
			return fmt.Errorf("discard: unexpected EOF")
		}
		remaining -= read
	}
	return nil
}
