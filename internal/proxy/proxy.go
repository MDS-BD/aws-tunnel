package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/opensearch-project/opensearch-go/v4/signer"
	"github.com/opensearch-project/opensearch-go/v4/signer/awsv2"
)

// New returns a reverse proxy that SigV4-signs every request for host and sends it through tunnelAddr.
//
// Requests keep the real domain name for Host, SNI and signature, only the TCP dial is redirected to the
// tunnel: the TLS certificate is verified and the signature matches without any host override.
func New(cfg aws.Config, host, tunnelAddr string) (http.Handler, error) {
	return newHandler(cfg, host, tunnelAddr, &tls.Config{ServerName: host})
}

func newHandler(cfg aws.Config, host, tunnelAddr string, tlsConfig *tls.Config) (http.Handler, error) {
	s, err := awsv2.NewSigner(cfg)
	if err != nil {
		return nil, fmt.Errorf("create signer: %w", err)
	}

	dialer := &net.Dialer{}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, tunnelAddr)
	}
	transport.TLSClientConfig = tlsConfig

	target := &url.URL{Scheme: "https", Host: host}
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			// Drop client-side auth so it cannot clash with our signature.
			r.Out.Header.Del("Authorization")
			for h := range r.Out.Header {
				if strings.HasPrefix(h, "X-Amz-") {
					r.Out.Header.Del(h)
				}
			}
		},
		Transport: &signingTransport{base: transport, signer: s},
	}, nil
}

type signingTransport struct {
	base   http.RoundTripper
	signer signer.Signer
}

func (t *signingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if err := t.signer.SignRequest(r); err != nil {
		return nil, fmt.Errorf("sign request: %w", err)
	}
	return t.base.RoundTrip(r)
}
