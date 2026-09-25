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
	// IAM auth tokens are valid for 15 minutes: print a fresh one well before that.
	iamTokenRefresh   = 10 * time.Minute
	instanceKeyPrefix = "instance:"
	shutdownTimeout   = 5 * time.Second
)

// Globals are the flags shared by every subcommand.
type Globals struct {
	CacheTTL  time.Duration    `default:"24h" env:"AWS_TUNNEL_CACHE_TTL" help:"How long cached AWS listings stay fresh."`
	LocalPort int              `default:"0" env:"AWS_TUNNEL_LOCAL_PORT" help:"Local port of the SSM tunnel (0 = random)." name:"local"`
	Profile   string           `env:"AWS_PROFILE" help:"AWS shared config profile."`
	Refresh   bool             `help:"Ignore the cache and fetch listings from AWS." short:"r"`
	Region    string           `default:"eu-west-1" env:"AWS_REGION" help:"AWS region."`
	Target    string           `env:"AWS_TUNNEL_TARGET" help:"EC2 instance ID to tunnel through; skips the instance picker." short:"t"`
	Verbose   bool             `help:"Verbose logging." short:"v"`
	Version   kong.VersionFlag `help:"Print the version and exit."`
}

type CLI struct {
	Globals

	APIGateway  APIGatewayCmd  `cmd:"" help:"Local proxy to a private API Gateway REST API, optionally SigV4-signed." name:"apigateway"`
	EKS         EKSCmd         `cmd:"" help:"kubectl access to a private EKS API endpoint, no exec plugin needed." name:"eks"`
	ElastiCache ElastiCacheCmd `cmd:"" help:"Tunnel to a private ElastiCache Redis/Valkey cache, optionally with IAM auth tokens." name:"elasticache"`
	Opensearch  OpensearchCmd  `cmd:"" help:"SigV4-signing proxy to a private OpenSearch domain."`
	RDS         RDSCmd         `cmd:"" help:"Tunnel to a private RDS/Aurora database, optionally with IAM auth tokens." name:"rds"`
	Shell       ShellCmd       `cmd:"" help:"Interactive shell on an SSM-managed instance."`
	Tunnel      TunnelCmd      `cmd:"" help:"Forward a local port to any host:port the instance can reach (like gossm fwdrem)."`
}

// Session is what every subcommand runs with: AWS config and the per-account cache.
type Session struct {
	Globals

	Cfg   aws.Config
	Store *cache.Store
}

func Main(version string) {
	var cli CLI
	kctx := kong.Parse(&cli,
		kong.Name("aws-tunnel"),
		kong.Description("SSM tunnels to private AWS services, no bastion, CLI or plugin needed."),
		kong.Vars{"version": version},
	)

	log := newLogger(cli.Verbose)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx = log.WithContext(ctx)

	err := func() error {
		s, err := newSession(ctx, cli.Globals)
		if err != nil {
			return err
		}
		kctx.BindTo(ctx, (*context.Context)(nil))
		return kctx.Run(s)
	}()
	if err != nil {
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

func newSession(ctx context.Context, g Globals) (*Session, error) {
	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(g.Region)}
	if g.Profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(g.Profile))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	accountID, err := cloud.AccountID(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("check credentials (authenticate first): %w", err)
	}

	store, err := cache.Open(accountID, cfg.Region)
	if err != nil {
		return nil, err
	}
	zerolog.Ctx(ctx).Debug().Str("path", store.Path()).Msg("cache opened")

	return &Session{Globals: g, Cfg: cfg, Store: store}, nil
}

// SaveCache persists listings and usage; failing to do so only costs a refetch next time.
func (s *Session) SaveCache(ctx context.Context) {
	if err := s.Store.Save(); err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("could not save cache")
	}
}

func (s *Session) PickTarget(ctx context.Context) (string, error) {
	if s.Target != "" {
		s.Store.Touch(instanceKeyPrefix + s.Target)
		return s.Target, nil
	}

	listing := cache.Listing[cloud.Instance](s.Store, "instances")
	instances, fetched, err := cache.Get(ctx, listing, s.CacheTTL, s.Refresh,
		func(ctx context.Context) ([]cloud.Instance, error) { return cloud.ListInstances(ctx, s.Cfg) })
	if err != nil {
		return "", err
	}
	zerolog.Ctx(ctx).Debug().Bool("fetched", fetched).Int("count", len(instances)).Msg("instances loaded")

	instanceKey := func(i cloud.Instance) string { return instanceKeyPrefix + i.ID }
	cache.Rank(s.Store, instances, instanceKey, func(i cloud.Instance) string { return i.Name })

	i, err := picker.Pick(instances, "ssm target", header("select an SSM target instance", listing.FetchedAt),
		instanceLabel)
	if err != nil {
		return "", err
	}
	if !i.SupportsMux() {
		zerolog.Ctx(ctx).Warn().Str("agentVersion", i.AgentVersion).
			Msg("SSM agent older than 3.0.196.0: connections are served one at a time, update it with AWS-UpdateSSMAgent")
	}
	s.Store.Touch(instanceKey(i))
	return i.ID, nil
}

// OpenTunnel forwards a local port to host:remotePort through target, returning the local address.
func (s *Session) OpenTunnel(ctx context.Context, target, host string, remotePort int) (string, <-chan error, error) {
	localPort := s.LocalPort
	if localPort == 0 {
		port, err := freePort()
		if err != nil {
			return "", nil, err
		}
		localPort = port
	}

	zerolog.Ctx(ctx).Info().Str("host", host).Int("remotePort", remotePort).Str("target", target).
		Int("localPort", localPort).Msg("opening SSM tunnel")
	done, err := tunnel.Start(ctx, s.Cfg, tunnel.Options{
		Host:       host,
		LocalPort:  localPort,
		RemotePort: remotePort,
		Target:     target,
	})
	if err != nil {
		return "", nil, err
	}
	return net.JoinHostPort("localhost", strconv.Itoa(localPort)), done, nil
}

// ServeProxy tunnels to opts.Host:remotePort through target and serves a local HTTP proxy to it until ctx ends.
func (s *Session) ServeProxy(ctx context.Context, target string, remotePort, proxyPort int, opts proxy.Options) error {
	tunnelAddr, tunnelDone, err := s.OpenTunnel(ctx, target, opts.Host, remotePort)
	if err != nil {
		return err
	}

	opts.TunnelAddr = tunnelAddr
	handler, err := proxy.New(s.Cfg, opts)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              net.JoinHostPort("localhost", strconv.Itoa(proxyPort)),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	srvDone := make(chan error, 1)
	go func() { srvDone <- srv.ListenAndServe() }()

	log := zerolog.Ctx(ctx)
	kind := "proxy"
	if opts.Service != "" {
		kind = "SigV4 proxy (" + opts.Service + ")"
	}
	log.Info().Msg(kind + " ready on http://" + srv.Addr + " — Ctrl+C to stop")

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

// WaitTunnel blocks until ctx ends or the tunnel closes. When token is set it prints a fresh IAM token now and
// every iamTokenRefresh, since tokens expire after 15 minutes and only matter for new connections.
func (s *Session) WaitTunnel(ctx context.Context, tunnelDone <-chan error, token func(context.Context) (string, error)) error {
	log := zerolog.Ctx(ctx)
	printToken := func() error {
		t, err := token(ctx)
		if err != nil {
			return fmt.Errorf("build IAM auth token: %w", err)
		}
		log.Info().Msg("IAM token (valid 15 minutes, new connections only):\n" + t)
		return nil
	}

	var refresh <-chan time.Time
	if token != nil {
		if err := printToken(); err != nil {
			return err
		}
		ticker := time.NewTicker(iamTokenRefresh)
		defer ticker.Stop()
		refresh = ticker.C
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("shutting down…")
			return nil
		case err := <-tunnelDone:
			return fmt.Errorf("ssm tunnel closed: %w", err)
		case <-refresh:
			if err := printToken(); err != nil {
				log.Warn().Err(err).Msg("could not refresh IAM token")
			}
		}
	}
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

func freePort() (int, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("find free port: %w", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
