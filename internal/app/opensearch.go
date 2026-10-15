package app

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/MDS-BD/aws-tunnel/internal/cache"
	"github.com/MDS-BD/aws-tunnel/internal/cloud"
	"github.com/MDS-BD/aws-tunnel/internal/picker"
	"github.com/MDS-BD/aws-tunnel/internal/proxy"
)

const domainKeyPrefix = "domain:"

type OpensearchCmd struct {
	Open       bool   `env:"AWS_TUNNEL_OPENSEARCH_OPEN" help:"Open OpenSearch Dashboards in the browser once the proxy is ready."`
	Host       string `env:"AWS_TUNNEL_OPENSEARCH_HOST" help:"OpenSearch endpoint; skips the domain picker."`
	ProxyPort  int    `default:"8080" env:"AWS_TUNNEL_PROXY_PORT" help:"Base port of the SigV4 signing proxy; the next free one is used when busy."`
	RemotePort int    `default:"443" help:"Port of the OpenSearch endpoint." name:"remote"`
}

func (c *OpensearchCmd) Run(ctx context.Context, s *Session) error {
	host, err := c.pickDomain(ctx, s)
	if err != nil {
		return err
	}
	target, err := s.PickTarget(ctx)
	if err != nil {
		return err
	}
	s.SaveCache(ctx)

	openPath := ""
	if c.Open {
		openPath = "/_dashboards"
	}
	ln, err := ListenFrom(c.ProxyPort)
	if err != nil {
		return err
	}
	return s.ServeProxy(ctx, target, c.RemotePort, ln, proxy.Options{Host: host, Service: "es"}, openPath)
}

func (c *OpensearchCmd) pickDomain(ctx context.Context, s *Session) (string, error) {
	if c.Host != "" {
		s.Store.Touch(domainKeyPrefix + c.Host)
		return c.Host, nil
	}

	listing := cache.Listing[cloud.Domain](s.Store, "opensearch-domains")
	domains, fetched, err := cache.Get(ctx, listing, s.CacheTTL, s.Refresh,
		func(ctx context.Context) ([]cloud.Domain, error) { return cloud.ListDomains(ctx, s.Cfg) })
	if err != nil {
		return "", err
	}
	zerolog.Ctx(ctx).Debug().Bool("fetched", fetched).Int("count", len(domains)).Msg("domains loaded")

	domainKey := func(d cloud.Domain) string { return domainKeyPrefix + d.Endpoint }
	cache.Rank(s.Store, domains, domainKey, func(d cloud.Domain) string { return d.Name })

	d, err := picker.Pick(domains, "domain", header("select a target domain", listing.FetchedAt),
		func(d cloud.Domain) string { return d.Name + "  (" + d.EngineVersion + ")" })
	if err != nil {
		return "", err
	}
	s.Store.Touch(domainKey(d))
	return d.Endpoint, nil
}
