package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog"
	"go.uber.org/zap"

	"github.com/MDS-BD/aws-tunnel/internal/cache"
	"github.com/MDS-BD/aws-tunnel/internal/cloud"
	"github.com/MDS-BD/aws-tunnel/internal/picker"
	"github.com/MDS-BD/aws-tunnel/internal/proxy"
	"github.com/MDS-BD/aws-tunnel/internal/tunnel"
)

const (
	domainKeyPrefix   = "domain:"
	instanceKeyPrefix = "instance:"
	shutdownTimeout   = 5 * time.Second
)

type CLI struct {
	CacheTTL   time.Duration    `default:"24h" env:"AWS_TUNNEL_CACHE_TTL" help:"How long cached domain/instance lists stay fresh."`
	Host       string           `env:"AWS_TUNNEL_HOST" help:"OpenSearch endpoint; skips the domain picker."`
	LocalPort  int              `default:"0" env:"LOCAL_PORT" help:"Local port for the SSM tunnel (0 = random)." name:"local"`
	Profile    string           `env:"AWS_PROFILE" help:"AWS shared config profile."`
	ProxyPort  int              `default:"8080" env:"PROXY_PORT" help:"Port of the SigV4 signing proxy."`
	Refresh    bool             `help:"Ignore the cache and fetch domains and instances from AWS." short:"r"`
	Region     string           `default:"eu-west-1" env:"REGION,AWS_REGION" help:"AWS region."`
	RemotePort int              `default:"443" env:"REMOTE_PORT" help:"Port of the OpenSearch endpoint." name:"remote"`
	Target     string           `env:"AWS_TUNNEL_TARGET" help:"EC2 instance ID to tunnel through; skips the instance picker." short:"t"`
	Verbose    bool             `help:"Verbose logging." short:"v"`
	Version    kong.VersionFlag `help:"Print the version and exit."`
}

func Main(version string) {
	var cli CLI
	kong.Parse(&cli,
		kong.Name("aws-tunnel"),
		kong.Description("SSM tunnels to private AWS services, no bastion, CLI or plugin needed."),
		kong.Vars{"version": version},
	)

	log := newLogger(cli.Verbose)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(log.WithContext(ctx), cli); err != nil {
		if errors.Is(err, picker.ErrAborted) || errors.Is(err, context.Canceled) {
			os.Exit(130)
		}
		log.Error().Err(err).Msg("aws-tunnel failed")
		os.Exit(1)
	}
}

func newLogger(verbose bool) zerolog.Logger {
	level := zerolog.InfoLevel
	if verbose {
		level = zerolog.DebugLevel
		// The SSM client logs through zap's global logger, a no-op unless replaced.
		if z, err := zap.NewDevelopment(); err == nil {
			zap.ReplaceGlobals(z)
		}
	}
	return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.TimeOnly}).
		Level(level).With().Timestamp().Logger()
}

func run(ctx context.Context, cli CLI) error {
	log := zerolog.Ctx(ctx)

	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cli.Region)}
	if cli.Profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(cli.Profile))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	accountID, err := cloud.AccountID(ctx, cfg)
	if err != nil {
		return fmt.Errorf("check credentials (authenticate first): %w", err)
	}

	store, err := cache.Open(accountID, cfg.Region)
	if err != nil {
		return err
	}
	log.Debug().Str("path", store.Path()).Msg("cache opened")

	host, err := pickHost(ctx, cfg, store, cli)
	if err != nil {
		return err
	}
	target, err := pickTarget(ctx, cfg, store, cli)
	if err != nil {
		return err
	}
	if err := store.Save(); err != nil {
		log.Warn().Err(err).Msg("could not save cache")
	}

	return serve(ctx, cfg, cli, host, target)
}

func pickHost(ctx context.Context, cfg aws.Config, store *cache.Store, cli CLI) (string, error) {
	if cli.Host != "" {
		store.Touch(domainKeyPrefix + cli.Host)
		return cli.Host, nil
	}

	domains, fetched, err := cache.Get(ctx, store.Domains(), cli.CacheTTL, cli.Refresh,
		func(ctx context.Context) ([]cloud.Domain, error) { return cloud.ListDomains(ctx, cfg) })
	if err != nil {
		return "", err
	}
	zerolog.Ctx(ctx).Debug().Bool("fetched", fetched).Int("count", len(domains)).Msg("domains loaded")

	domainKey := func(d cloud.Domain) string { return domainKeyPrefix + d.Endpoint }
	cache.Rank(store, domains, domainKey, func(d cloud.Domain) string { return d.Name })

	d, err := picker.Pick(domains, "domain", header("select a target domain", store.Domains().FetchedAt),
		func(d cloud.Domain) string { return d.Name + "  (" + d.EngineVersion + ")" })
	if err != nil {
		return "", err
	}
	store.Touch(domainKey(d))
	return d.Endpoint, nil
}

func pickTarget(ctx context.Context, cfg aws.Config, store *cache.Store, cli CLI) (string, error) {
	if cli.Target != "" {
		store.Touch(instanceKeyPrefix + cli.Target)
		return cli.Target, nil
	}

	instances, fetched, err := cache.Get(ctx, store.Instances(), cli.CacheTTL, cli.Refresh,
		func(ctx context.Context) ([]cloud.Instance, error) { return cloud.ListInstances(ctx, cfg) })
	if err != nil {
		return "", err
	}
	zerolog.Ctx(ctx).Debug().Bool("fetched", fetched).Int("count", len(instances)).Msg("instances loaded")

	instanceKey := func(i cloud.Instance) string { return instanceKeyPrefix + i.ID }
	cache.Rank(store, instances, instanceKey, func(i cloud.Instance) string { return i.Name })

	i, err := picker.Pick(instances, "ssm target", header("select an SSM target instance", store.Instances().FetchedAt),
		instanceLabel)
	if err != nil {
		return "", err
	}
	if !i.SupportsMux() {
		zerolog.Ctx(ctx).Warn().Str("agentVersion", i.AgentVersion).
			Msg("SSM agent older than 3.0.196.0: connections are served one at a time, update it with AWS-UpdateSSMAgent")
	}
	store.Touch(instanceKey(i))
	return i.ID, nil
}

func instanceLabel(i cloud.Instance) string {
	label := i.Name + "  " + i.ID + "  " + i.Platform + "  agent " + i.AgentVersion
	if !i.SupportsMux() {
		label += " (no multiplexing)"
	}
	return label
}

func header(title string, fetchedAt time.Time) string {
	return title + " — cached " + fetchedAt.Format(time.DateTime) + ", --refresh to reload (Esc to cancel)"
}

func serve(ctx context.Context, cfg aws.Config, cli CLI, host, target string) error {
	log := zerolog.Ctx(ctx)

	localPort := cli.LocalPort
	if localPort == 0 {
		port, err := freePort()
		if err != nil {
			return err
		}
		localPort = port
	}

	log.Info().Str("host", host).Str("target", target).Int("localPort", localPort).Msg("opening SSM tunnel")
	tunnelDone, err := tunnel.Start(ctx, cfg, tunnel.Options{
		Host:       host,
		LocalPort:  localPort,
		RemotePort: cli.RemotePort,
		Target:     target,
	})
	if err != nil {
		return err
	}

	handler, err := proxy.New(cfg, host, net.JoinHostPort("localhost", strconv.Itoa(localPort)))
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              net.JoinHostPort("localhost", strconv.Itoa(cli.ProxyPort)),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	srvDone := make(chan error, 1)
	go func() { srvDone <- srv.ListenAndServe() }()

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

func freePort() (int, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("find free port: %w", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
