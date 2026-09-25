package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/MDS-BD/aws-tunnel/internal/cache"
	"github.com/MDS-BD/aws-tunnel/internal/cloud"
	"github.com/MDS-BD/aws-tunnel/internal/picker"
	"github.com/MDS-BD/aws-tunnel/internal/proxy"
)

const domainKeyPrefix = "domain:"

type OpensearchCmd struct {
	Host       string `env:"AWS_TUNNEL_OPENSEARCH_HOST" help:"OpenSearch endpoint; skips the domain picker."`
	ProxyPort  int    `default:"8080" env:"AWS_TUNNEL_PROXY_PORT" help:"Port of the SigV4 signing proxy."`
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

	tunnelAddr, tunnelDone, err := s.OpenTunnel(ctx, target, host, c.RemotePort)
	if err != nil {
		return err
	}

	handler, err := proxy.New(s.Cfg, host, tunnelAddr)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              net.JoinHostPort("localhost", strconv.Itoa(c.ProxyPort)),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	srvDone := make(chan error, 1)
	go func() { srvDone <- srv.ListenAndServe() }()

	log := zerolog.Ctx(ctx)
	log.Info().Msg("SigV4 proxy ready on http://" + srv.Addr + " — Ctrl+C to stop")

	select {
	case <-ctx.Done():
	case err = <-tunnelDone:
		err = fmt.Errorf("ssm tunnel closed: %w", err)
	case err = <-srvDone:
		err = fmt.Errorf("proxy server: %w", err)
	}

	log.Info().Msg("shutting down…")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	return err
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
