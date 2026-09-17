package ssh

import (
	"bufio"
	"context"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type ForwardSpec struct {
	LocalAddr  string
	RemoteAddr string
}

type ForwardHandle struct {
	Listener net.Listener
	Done     <-chan error
	Stop     func() error
}

func StartLocalForward(ctx context.Context, client *ssh.Client, spec ForwardSpec) (*ForwardHandle, error) {
	ln, err := net.Listen("tcp", spec.LocalAddr)
	if err != nil { return nil, err }
	done := make(chan error, 1)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil { if ctx.Err() != nil { done <- nil } else { done <- err }; return }
			go proxyConn(ctx, conn, func() (net.Conn, error) { return client.Dial("tcp", spec.RemoteAddr) })
		}
	}()
	return &ForwardHandle{Listener: ln, Done: done, Stop: func() error { cancel(); return ln.Close() }}, nil
}

func StartRemoteForward(ctx context.Context, client *ssh.Client, spec ForwardSpec) (*ForwardHandle, error) {
	ln, err := client.Listen("tcp", spec.RemoteAddr)
	if err != nil { return nil, err }
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil { if ctx.Err() != nil { done <- nil } else { done <- err }; return }
			go proxyConn(ctx, conn, func() (net.Conn, error) { return net.Dial("tcp", spec.LocalAddr) })
		}
	}()
	return &ForwardHandle{Listener: ln, Done: done, Stop: func() error { cancel(); return ln.Close() }}, nil
}

func StartDynamicForward(ctx context.Context, client *ssh.Client, localAddr string) (*ForwardHandle, error) {
	ln, err := net.Listen("tcp", localAddr)
	if err != nil { return nil, err }
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil { if ctx.Err() != nil { done <- nil } else { done <- err }; return }
			go handleSOCKS5(ctx, client, conn)
		}
	}()
	return &ForwardHandle{Listener: ln, Done: done, Stop: func() error { cancel(); return ln.Close() }}, nil
}

func proxyConn(ctx context.Context, src net.Conn, dial func() (net.Conn, error)) {
	defer src.Close()
	dst, err := dial(); if err != nil { return }
	defer dst.Close()
	var wg sync.WaitGroup; wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(dst, src); if tcp, ok := dst.(*net.TCPConn); ok { _ = tcp.CloseWrite() } }()
	go func() { defer wg.Done(); _, _ = io.Copy(src, dst); if tcp, ok := src.(*net.TCPConn); ok { _ = tcp.CloseWrite() } }()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select { case <-ctx.Done(): _ = src.Close(); _ = dst.Close(); case <-done: }
}

func handleSOCKS5(ctx context.Context, client *ssh.Client, conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	ver, err := br.ReadByte(); if err != nil || ver != 5 { return }
	n, err := br.ReadByte(); if err != nil { return }
	methods := make([]byte, int(n)); if _, err = io.ReadFull(br, methods); err != nil { return }
	if _, err = conn.Write([]byte{5, 0}); err != nil { return }
	header := make([]byte, 4); if _, err = io.ReadFull(br, header); err != nil || header[0] != 5 || header[1] != 1 { return }
	atyp := header[3]
	var host string
	var portBytes [2]byte
	switch atyp {
	case 1:
		b := make([]byte, 4); if _, err = io.ReadFull(br, b); err != nil { return }; host = net.IP(b).String()
	case 3:
		l, e := br.ReadByte(); if e != nil { return }; b := make([]byte, int(l)); if _, e = io.ReadFull(br, b); e != nil { return }; host = string(b)
	case 4:
		b := make([]byte, 16); if _, err = io.ReadFull(br, b); err != nil { return }; host = net.IP(b).String()
	default: return
	}
	if _, err = io.ReadFull(br, portBytes[:]); err != nil { return }
	port := strconv.Itoa(int(portBytes[0])<<8 | int(portBytes[1]))
	dst, err := client.Dial("tcp", net.JoinHostPort(host, port)); if err != nil { _, _ = conn.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0}); return }
	defer dst.Close()
	if _, err = conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}); err != nil { return }
	if d, ok := ctx.Deadline(); ok { _ = conn.SetDeadline(d) }
	var wg sync.WaitGroup; wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(dst, br) }()
	go func() { defer wg.Done(); _, _ = io.Copy(conn, dst) }()
	wg.Wait()
}

var _ = time.Second
