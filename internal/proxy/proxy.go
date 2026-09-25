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

type Options struct {
	// Host is the real service host name, used for Host, SNI and the signature.
	Host string
	// PathPrefix is prepended to every request path, e.g. an API Gateway stage.
	PathPrefix string
	// Service is the SigV4 signing name ("es", "execute-api"...); empty forwards requests unsigned.
	Service string
	// TunnelAddr is where connections are dialled instead of Host.
	TunnelAddr string
}

// New returns a reverse proxy to https://Host that sends every request through TunnelAddr.
//
// Requests keep the real host name for Host, SNI and signature, only the TCP dial is redirected to the
// tunnel: the TLS certificate is verified and the signature matches without any host override.
func New(cfg aws.Config, opts Options) (http.Handler, error) {
	return newHandler(cfg, opts, &tls.Config{ServerName: opts.Host})
}

func newHandler(cfg aws.Config, opts Options, tlsConfig *tls.Config) (http.Handler, error) {
	dialer := &net.Dialer{}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, opts.TunnelAddr)
	}
	transport.TLSClientConfig = tlsConfig

	var roundTripper http.RoundTripper = transport
	if opts.Service != "" {
		s, err := awsv2.NewSignerWithService(cfg, opts.Service)
		if err != nil {
			return nil, fmt.Errorf("create signer: %w", err)
		}
		roundTripper = &signingTransport{base: transport, signer: s}
	}

	target := &url.URL{Scheme: "https", Host: opts.Host, Path: opts.PathPrefix}
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			if opts.Service == "" {
				return
			}
			// Drop client-side auth so it cannot clash with our signature.
			r.Out.Header.Del("Authorization")
			for h := range r.Out.Header {
				if strings.HasPrefix(h, "X-Amz-") {
					r.Out.Header.Del(h)
				}
			}
		},
		Transport: roundTripper,
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
