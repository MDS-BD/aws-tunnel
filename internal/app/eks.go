package app

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"

	"github.com/MDS-BD/aws-tunnel/internal/cache"
	"github.com/MDS-BD/aws-tunnel/internal/cloud"
	"github.com/MDS-BD/aws-tunnel/internal/picker"
	"github.com/MDS-BD/aws-tunnel/internal/proxy"
)

const clusterKeyPrefix = "eks:"

type EKSCmd struct {
	Cluster   string `env:"AWS_TUNNEL_EKS_CLUSTER" help:"Cluster name; skips the cluster picker."`
	ProxyPort int    `default:"8443" env:"AWS_TUNNEL_EKS_PROXY_PORT" help:"Port of the local API server proxy."`
}

func (c *EKSCmd) Run(ctx context.Context, s *Session) error {
	cluster, err := c.pickCluster(ctx, s)
	if err != nil {
		return err
	}
	host, err := cluster.Host()
	if err != nil {
		return err
	}
	target, err := s.PickTarget(ctx)
	if err != nil {
		return err
	}
	s.SaveCache(ctx)

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(cluster.CAData) {
		return fmt.Errorf("cluster %s: no usable certificate authority", cluster.Name)
	}
	tokens := &tokenCache{fetch: func(ctx context.Context) (token.Token, error) {
		return cloud.ClusterToken(ctx, s.Cfg, cluster.Name)
	}}
	// Fail now, not on the first kubectl call, when credentials cannot sign a token.
	if _, err := tokens.Get(ctx); err != nil {
		return err
	}

	kubeconfig, err := writeKubeconfig(cluster.Name, c.ProxyPort)
	if err != nil {
		return err
	}
	defer os.Remove(kubeconfig)

	log := zerolog.Ctx(ctx)
	log.Info().Msg("kubeconfig: export KUBECONFIG=" + kubeconfig + "  (removed on exit)")
	if cluster.PublicAccess {
		log.Warn().Msg("this cluster also has a public endpoint: the tunnel is not needed from allowed networks")
	}

	// The proxy verifies the API server against the cluster CA and adds the IAM token: kubectl talks plain HTTP
	// to localhost and needs no aws CLI or exec plugin.
	return s.ServeProxy(ctx, target, 443, c.ProxyPort, proxy.Options{
		BearerToken: tokens.Get,
		Host:        host,
		RootCAs:     roots,
	})
}

func (c *EKSCmd) pickCluster(ctx context.Context, s *Session) (cloud.Cluster, error) {
	if c.Cluster != "" {
		s.Store.Touch(clusterKeyPrefix + c.Cluster)
		return cloud.DescribeCluster(ctx, s.Cfg, c.Cluster)
	}

	listing := cache.Listing[cloud.Cluster](s.Store, "eks-clusters")
	clusters, fetched, err := cache.Get(ctx, listing, s.CacheTTL, s.Refresh,
		func(ctx context.Context) ([]cloud.Cluster, error) { return cloud.ListClusters(ctx, s.Cfg) })
	if err != nil {
		return cloud.Cluster{}, err
	}
	zerolog.Ctx(ctx).Debug().Bool("fetched", fetched).Int("count", len(clusters)).Msg("clusters loaded")

	clusterKey := func(c cloud.Cluster) string { return clusterKeyPrefix + c.Name }
	cache.Rank(s.Store, clusters, clusterKey, func(c cloud.Cluster) string { return c.Name })

	cluster, err := picker.Pick(clusters, "cluster", header("select a private EKS cluster", listing.FetchedAt),
		func(c cloud.Cluster) string { return c.Name + "  k8s " + c.Version })
	if err != nil {
		return cloud.Cluster{}, err
	}
	s.Store.Touch(clusterKey(cluster))
	return cluster, nil
}

// writeKubeconfig writes a kubeconfig pointing at the local proxy. The proxy handles TLS and auth, so there are
// no credentials in it.
func writeKubeconfig(clusterName string, proxyPort int) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("user cache dir: %w", err)
	}
	path := filepath.Join(dir, "aws-tunnel", "kubeconfig-"+clusterName+"-"+strconv.Itoa(proxyPort))

	cfg := clientcmdapi.NewConfig()
	cfg.Clusters[clusterName] = &clientcmdapi.Cluster{
		Server: "http://" + net.JoinHostPort("localhost", strconv.Itoa(proxyPort)),
	}
	cfg.AuthInfos[clusterName] = &clientcmdapi.AuthInfo{}
	cfg.Contexts[clusterName] = &clientcmdapi.Context{AuthInfo: clusterName, Cluster: clusterName}
	cfg.CurrentContext = clusterName

	if err := clientcmd.WriteToFile(*cfg, path); err != nil {
		return "", fmt.Errorf("write kubeconfig: %w", err)
	}
	return path, nil
}

// tokenCache hands out the current EKS token, fetching a new one shortly before it expires.
type tokenCache struct {
	fetch func(context.Context) (token.Token, error)
	mu    sync.Mutex
	token token.Token
}

// tokenRefreshMargin renews tokens early, so a request signed now is still valid when it reaches the API server.
const tokenRefreshMargin = time.Minute

func (t *tokenCache) Get(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token.Token != "" && time.Until(t.token.Expiration) > tokenRefreshMargin {
		return t.token.Token, nil
	}
	tok, err := t.fetch(ctx)
	if err != nil {
		return "", err
	}
	if tok.Token == "" {
		return "", errors.New("empty EKS token")
	}
	t.token = tok
	return tok.Token, nil
}
