package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	handler, err := newHandler(cfg, "example.com", upstream.Listener.Addr().String(),
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
