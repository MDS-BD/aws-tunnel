package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"github.com/MDS-BD/aws-tunnel/internal/cache"
	"github.com/MDS-BD/aws-tunnel/internal/cloud"
	"github.com/MDS-BD/aws-tunnel/internal/picker"
	"github.com/MDS-BD/aws-tunnel/internal/proxy"
)

const apiKeyPrefix = "apigateway:"

type APIGatewayCmd struct {
	API       string `env:"AWS_TUNNEL_API" help:"REST API ID; skips the API picker (requires --stage)." name:"api"`
	ProxyPort int    `default:"8080" env:"AWS_TUNNEL_PROXY_PORT" help:"Base port of the local proxy; the next free one is used when busy."`
	Sign      bool   `help:"SigV4-sign requests, for APIs using AWS_IAM authorization."`
	Stage     string `help:"Stage to call; requests to http://localhost:PORT/x go to /STAGE/x."`
}

func (c *APIGatewayCmd) Run(ctx context.Context, s *Session) error {
	api, err := c.pickAPI(ctx, s)
	if err != nil {
		return err
	}
	target, err := s.PickTarget(ctx)
	if err != nil {
		return err
	}
	s.SaveCache(ctx)

	opts := proxy.Options{Host: api.Host(s.Cfg.Region), PathPrefix: "/" + api.Stage}
	if c.Sign {
		opts.Service = "execute-api"
	}
	ln, err := ListenFrom(c.ProxyPort)
	if err != nil {
		return err
	}
	return s.ServeProxy(ctx, target, 443, ln, opts, "")
}

func (c *APIGatewayCmd) pickAPI(ctx context.Context, s *Session) (cloud.API, error) {
	if c.API != "" {
		if c.Stage == "" {
			return cloud.API{}, fmt.Errorf("--stage is required with --api")
		}
		api := cloud.API{ID: c.API, Stage: strings.Trim(c.Stage, "/")}
		s.Store.Touch(apiKey(api))
		return api, nil
	}

	listing := cache.Listing[cloud.API](s.Store, "apigateway-private-apis")
	apis, fetched, err := cache.Get(ctx, listing, s.CacheTTL, s.Refresh,
		func(ctx context.Context) ([]cloud.API, error) { return cloud.ListPrivateAPIs(ctx, s.Cfg) })
	if err != nil {
		return cloud.API{}, err
	}
	zerolog.Ctx(ctx).Debug().Bool("fetched", fetched).Int("count", len(apis)).Msg("private APIs loaded")

	cache.Rank(s.Store, apis, apiKey, func(a cloud.API) string { return a.Name + "/" + a.Stage })

	api, err := picker.Pick(apis, "api", header("select a private API stage", listing.FetchedAt),
		func(a cloud.API) string { return a.Name + "  /" + a.Stage + "  " + a.ID })
	if err != nil {
		return cloud.API{}, err
	}
	s.Store.Touch(apiKey(api))
	return api, nil
}

func apiKey(a cloud.API) string { return apiKeyPrefix + a.ID + "/" + a.Stage }
