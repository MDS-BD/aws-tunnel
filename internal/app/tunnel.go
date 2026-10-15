package app

import (
	"context"
	"strconv"

	"github.com/rs/zerolog"
)

// TunnelCmd is a plain TCP port forward, like `gossm fwdrem`: localhost:--local reaches --host:--remote
// through the target instance. The instance resolves --host, so VPC-private DNS names work.
type TunnelCmd struct {
	Host      string `env:"AWS_TUNNEL_HOST" help:"Remote host, resolved by the instance; omit to reach a port on the instance itself."`
	Remote    int    `env:"AWS_TUNNEL_REMOTE_PORT" help:"Remote port." required:""`
	LocalPort int    `default:"8080" env:"AWS_TUNNEL_TUNNEL_LOCAL_PORT" help:"Base local port; the next free one is used when busy. --local overrides it." name:"local-base"`
}

func (c *TunnelCmd) Run(ctx context.Context, s *Session) error {
	target, err := s.PickTarget(ctx)
	if err != nil {
		return err
	}
	s.SaveCache(ctx)

	tunnelAddr, tunnelDone, err := s.OpenTunnel(ctx, target, c.Host, c.Remote, c.LocalPort)
	if err != nil {
		return err
	}

	destination := c.Host
	if destination == "" {
		destination = target
	}
	zerolog.Ctx(ctx).Info().
		Msg(tunnelAddr + " -> " + destination + ":" + strconv.Itoa(c.Remote) + " via " + target + " — Ctrl+C to stop")
	return s.WaitTunnel(ctx, tunnelDone, nil)
}
