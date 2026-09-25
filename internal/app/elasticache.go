package app

import (
	"context"
	"net"

	"github.com/rs/zerolog"

	"github.com/MDS-BD/aws-tunnel/internal/cache"
	"github.com/MDS-BD/aws-tunnel/internal/cloud"
	"github.com/MDS-BD/aws-tunnel/internal/picker"
)

const cacheKeyPrefix = "elasticache:"

type ElastiCacheCmd struct {
	IAM  bool   `help:"Print an IAM auth token to use as password, refreshed every 10 minutes." name:"iam"`
	User string `default:"default" help:"Cache user (RBAC / IAM-enabled user ID)."`
}

func (c *ElastiCacheCmd) Run(ctx context.Context, s *Session) error {
	ec, err := c.pickCache(ctx, s)
	if err != nil {
		return err
	}
	target, err := s.PickTarget(ctx)
	if err != nil {
		return err
	}
	s.SaveCache(ctx)

	log := zerolog.Ctx(ctx)
	if ec.ClusterMode {
		log.Warn().Msg("cluster mode enabled: MOVED/ASK redirects point at node addresses the tunnel does not forward, " +
			"use a client without cluster support (e.g. redis-cli without -c) for single-slot commands")
	}

	tunnelAddr, tunnelDone, err := s.OpenTunnel(ctx, target, ec.Host, ec.Port)
	if err != nil {
		return err
	}
	_, localPort, _ := net.SplitHostPort(tunnelAddr)

	log.Info().Msg("cache ready on " + tunnelAddr + " — Ctrl+C to stop")
	log.Info().Msg("connect with: " + cacheConnectHint(ec, localPort, c.User, c.IAM))

	if !c.IAM {
		return s.WaitTunnel(ctx, tunnelDone, nil)
	}
	return s.WaitTunnel(ctx, tunnelDone, func(ctx context.Context) (string, error) {
		return cloud.CacheIAMToken(ctx, s.Cfg, ec, c.User)
	})
}

func (c *ElastiCacheCmd) pickCache(ctx context.Context, s *Session) (cloud.Cache, error) {
	listing := cache.Listing[cloud.Cache](s.Store, "elasticache-caches")
	caches, fetched, err := cache.Get(ctx, listing, s.CacheTTL, s.Refresh,
		func(ctx context.Context) ([]cloud.Cache, error) { return cloud.ListCaches(ctx, s.Cfg) })
	if err != nil {
		return cloud.Cache{}, err
	}
	zerolog.Ctx(ctx).Debug().Bool("fetched", fetched).Int("count", len(caches)).Msg("caches loaded")

	cacheKey := func(c cloud.Cache) string { return cacheKeyPrefix + c.Host }
	cache.Rank(s.Store, caches, cacheKey, func(c cloud.Cache) string { return c.Name + " " + c.Role })

	ec, err := picker.Pick(caches, "cache", header("select a cache", listing.FetchedAt), cacheLabel)
	if err != nil {
		return cloud.Cache{}, err
	}
	s.Store.Touch(cacheKey(ec))
	return ec, nil
}

func cacheLabel(c cloud.Cache) string {
	label := c.Name + "  " + c.Role + "  " + c.Engine
	if c.Serverless {
		label += "  serverless"
	}
	if c.ClusterMode {
		label += "  cluster-mode"
	}
	if c.TLS {
		label += "  tls"
	}
	return label
}

// cacheConnectHint suggests a redis-cli command. Like RDS, the certificate names the cache host, not localhost:
// --insecure keeps encryption but skips host verification.
func cacheConnectHint(c cloud.Cache, port, user string, iam bool) string {
	hint := "redis-cli -h 127.0.0.1 -p " + port
	if c.TLS {
		hint += " --tls --insecure --sni " + c.Host
	}
	if iam || user != "default" {
		hint += " --user " + user + " --askpass"
	}
	return hint
}
