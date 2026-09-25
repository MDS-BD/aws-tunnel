package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func TestProxySignsAndTunnels(t *testing.T) {
	var got *http.Request
	var gotBody string
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = io.WriteString(w, "ok")
	}))
	defer upstream.Close()

	cfg := aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider("AKIDEXAMPLE", "secret", "token"),
		Region:      "eu-west-1",
	}
	pool := x509.NewCertPool()
	pool.AddCert(upstream.Certificate())
	// httptest certificates are valid for "example.com": use it as the domain name.
	handler, err := newHandler(cfg,
		Options{Host: "example.com", Service: "es", TunnelAddr: upstream.Listener.Addr().String()},
		&tls.Config{RootCAs: pool, ServerName: "example.com"})
	if err != nil {
		t.Fatalf("newHandler failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/_search?q=1", strings.NewReader(`{"query":{}}`))
	req.Header.Set("Authorization", "Basic abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if got.Host != "example.com" {
		t.Errorf("host mismatch: %s", got.Host)
	}
	if got.URL.RequestURI() != "/_search?q=1" {
		t.Errorf("uri mismatch: %s", got.URL.RequestURI())
	}
	if gotBody != `{"query":{}}` {
		t.Errorf("body mismatch: %s", gotBody)
	}
	auth := got.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/") || !strings.Contains(auth, "/eu-west-1/es/aws4_request") {
		t.Errorf("not signed for es: %s", auth)
	}
	if got.Header.Get("X-Amz-Security-Token") != "token" {
		t.Errorf("session token missing")
	}
}

func TestProxyUnsignedWithPathPrefix(t *testing.T) {
	var got *http.Request
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
	}))
	defer upstream.Close()

	pool := x509.NewCertPool()
	pool.AddCert(upstream.Certificate())
	handler, err := newHandler(aws.Config{},
		Options{Host: "example.com", PathPrefix: "/prod", TunnelAddr: upstream.Listener.Addr().String()},
		&tls.Config{RootCAs: pool, ServerName: "example.com"})
	if err != nil {
		t.Fatalf("newHandler failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/users/1?x=y", nil)
	req.Header.Set("Authorization", "Bearer abc")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got == nil {
		t.Fatalf("request did not reach upstream")
	}
	if got.URL.RequestURI() != "/prod/users/1?x=y" {
		t.Errorf("uri mismatch: %s", got.URL.RequestURI())
	}
	if got.Header.Get("Authorization") != "Bearer abc" {
		t.Errorf("unsigned proxy must keep client auth, got %q", got.Header.Get("Authorization"))
	}
}

func TestProxyBearerTokenAndRootCAs(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer upstream.Close()

	pool := x509.NewCertPool()
	pool.AddCert(upstream.Certificate())
	// Goes through New, not newHandler: RootCAs must reach the TLS config on its own.
	handler, err := New(aws.Config{}, Options{
		BearerToken: func(context.Context) (string, error) { return "k8s-aws-v1.abc", nil },
		Host:        "example.com",
		RootCAs:     pool,
		TunnelAddr:  upstream.Listener.Addr().String(),
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8443/api/v1/pods", nil)
	req.Header.Set("Authorization", "Bearer client")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if gotAuth != "Bearer k8s-aws-v1.abc" {
		t.Errorf("bearer token not injected: %q", gotAuth)
	}
}

// kubectl exec/attach/port-forward switch the connection to SPDY or WebSocket: the upgrade must pass through.
func TestProxyUpgrade(t *testing.T) {
	upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "SPDY/3.1" {
			http.Error(w, "no upgrade", http.StatusBadRequest)
			return
		}
		conn, rw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("Hijack failed: %v", err)
			return
		}
		defer conn.Close()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: SPDY/3.1\r\n\r\n")
		_ = rw.Flush()
		line, _ := rw.ReadString('\n')
		_, _ = rw.WriteString("echo:" + line)
		_ = rw.Flush()
	}))
	// Hijacking needs HTTP/1.1: keep the TLS server off HTTP/2.
	upstream.EnableHTTP2 = false
	upstream.StartTLS()
	defer upstream.Close()

	pool := x509.NewCertPool()
	pool.AddCert(upstream.Certificate())
	handler, err := New(aws.Config{}, Options{
		BearerToken: func(context.Context) (string, error) { return "t", nil },
		Host:        "example.com",
		RootCAs:     pool,
		TunnelAddr:  upstream.Listener.Addr().String(),
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	front := httptest.NewServer(handler)
	defer front.Close()

	conn, err := net.Dial("tcp", front.Listener.Addr().String())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()
	_, _ = io.WriteString(conn, "GET /api/v1/namespaces/default/pods/p/exec HTTP/1.1\r\nHost: localhost\r\n"+
		"Connection: Upgrade\r\nUpgrade: SPDY/3.1\r\n\r\n")

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected 101, got %d", resp.StatusCode)
	}
	_, _ = io.WriteString(conn, "hello\n")
	line, err := br.ReadString('\n')
	if err != nil || line != "echo:hello\n" {
		t.Errorf("upgraded stream mismatch: %q, %v", line, err)
	}
}

// kubectl logs -f and watches stream without Content-Length: each chunk must reach the client right away.
func TestProxyStreams(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "line1\n")
		w.(http.Flusher).Flush()
		<-release
		_, _ = io.WriteString(w, "line2\n")
	}))
	defer upstream.Close()

	pool := x509.NewCertPool()
	pool.AddCert(upstream.Certificate())
	handler, err := New(aws.Config{}, Options{Host: "example.com", RootCAs: pool, TunnelAddr: upstream.Listener.Addr().String()})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	front := httptest.NewServer(handler)
	defer front.Close()

	resp, err := http.Get(front.URL + "/api/v1/namespaces/default/pods/p/log?follow=true")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer resp.Body.Close()

	br := bufio.NewReader(resp.Body)
	done := make(chan string, 1)
	go func() {
		line, _ := br.ReadString('\n')
		done <- line
	}()
	select {
	case line := <-done:
		if line != "line1\n" {
			t.Errorf("first chunk mismatch: %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("first chunk held back by the proxy")
	}
	close(release)
}
