package app

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/rs/zerolog"

	"github.com/MDS-BD/aws-tunnel/internal/cache"
	"github.com/MDS-BD/aws-tunnel/internal/cloud"
	"github.com/MDS-BD/aws-tunnel/internal/picker"
)

const databaseKeyPrefix = "rds:"

type RDSCmd struct {
	Host      string `env:"AWS_TUNNEL_RDS_HOST" help:"Database endpoint; skips the database picker (requires --port)."`
	IAM       bool   `help:"Print an IAM auth token to use as password, refreshed every 10 minutes." name:"iam"`
	Port      int    `help:"Database port; defaults to the one RDS reports."`
	LocalPort int    `default:"50505" env:"AWS_TUNNEL_RDS_LOCAL_PORT" help:"Base local port; the next free one is used when busy. --local overrides it." name:"local-base"`
	User      string `help:"Database user; defaults to the master user."`
}

func (c *RDSCmd) Run(ctx context.Context, s *Session) error {
	db, err := c.pickDatabase(ctx, s)
	if err != nil {
		return err
	}
	target, err := s.PickTarget(ctx)
	if err != nil {
		return err
	}
	s.SaveCache(ctx)

	tunnelAddr, tunnelDone, err := s.OpenTunnel(ctx, target, db.Host, db.Port, c.LocalPort)
	if err != nil {
		return err
	}
	_, localPort, _ := net.SplitHostPort(tunnelAddr)

	log := zerolog.Ctx(ctx)
	user := c.User
	if user == "" {
		user = db.MasterUser
	}
	log.Info().Str("user", user).Msg("database ready on " + tunnelAddr + " — Ctrl+C to stop")
	if hint := connectHint(db.Engine, localPort, user, c.IAM); hint != "" {
		log.Info().Msg("connect with: " + hint)
	}

	if !c.IAM {
		return s.WaitTunnel(ctx, tunnelDone, nil)
	}
	if !db.IAMAuth {
		log.Warn().Msg("IAM database authentication looks disabled on this database: the token will be rejected")
	}
	// The token is signed for the real endpoint: RDS validates it against its own host name, not localhost.
	endpoint := net.JoinHostPort(db.Host, strconv.Itoa(db.Port))
	return s.WaitTunnel(ctx, tunnelDone, func(ctx context.Context) (string, error) {
		return auth.BuildAuthToken(ctx, endpoint, s.Cfg.Region, user, s.Cfg.Credentials)
	})
}

func (c *RDSCmd) pickDatabase(ctx context.Context, s *Session) (cloud.Database, error) {
	if c.Host != "" {
		if c.Port == 0 {
			return cloud.Database{}, fmt.Errorf("--port is required with --host")
		}
		s.Store.Touch(databaseKeyPrefix + c.Host)
		return cloud.Database{Host: c.Host, IAMAuth: true, Port: c.Port}, nil
	}

	listing := cache.Listing[cloud.Database](s.Store, "rds-databases")
	databases, fetched, err := cache.Get(ctx, listing, s.CacheTTL, s.Refresh,
		func(ctx context.Context) ([]cloud.Database, error) { return cloud.ListDatabases(ctx, s.Cfg) })
	if err != nil {
		return cloud.Database{}, err
	}
	zerolog.Ctx(ctx).Debug().Bool("fetched", fetched).Int("count", len(databases)).Msg("databases loaded")

	databaseKey := func(d cloud.Database) string { return databaseKeyPrefix + d.Host }
	cache.Rank(s.Store, databases, databaseKey, func(d cloud.Database) string { return d.Name + " " + d.Role })

	db, err := picker.Pick(databases, "database", header("select a database", listing.FetchedAt),
		func(d cloud.Database) string {
			return d.Name + "  " + d.Role + "  " + d.Engine + " " + d.EngineVersion + "  :" + strconv.Itoa(d.Port)
		})
	if err != nil {
		return cloud.Database{}, err
	}
	if c.Port != 0 {
		db.Port = c.Port
	}
	s.Store.Touch(databaseKey(db))
	return db, nil
}

// connectHint suggests a client command. TLS is required but not verified: the certificate names the RDS host,
// not localhost.
func connectHint(engine, port, user string, iam bool) string {
	switch {
	case strings.Contains(engine, "postgres"):
		return `psql "host=localhost port=` + port + ` user=` + user + ` dbname=postgres sslmode=require"`
	case strings.Contains(engine, "mysql"), strings.Contains(engine, "mariadb"):
		hint := "mysql -h 127.0.0.1 -P " + port + " -u " + user + " --ssl-mode=REQUIRED -p"
		if iam {
			// IAM tokens travel as cleartext passwords, protected by TLS.
			hint += " --enable-cleartext-plugin"
		}
		return hint
	}
	return ""
}
