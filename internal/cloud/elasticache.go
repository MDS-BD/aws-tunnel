package cloud

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
)

const (
	// emptyPayloadSHA256 is the SHA-256 of an empty body, what a presigned GET signs.
	emptyPayloadSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// Cache is a Redis/Valkey endpoint: a replication group primary/reader, or a serverless cache.
type Cache struct {
	ClusterMode bool   `json:"clusterMode"`
	Engine      string `json:"engine"`
	Host        string `json:"host"`
	// Name is the replication group ID or serverless cache name, what IAM tokens are scoped to.
	Name       string `json:"name"`
	Port       int    `json:"port"`
	Role       string `json:"role"`
	Serverless bool   `json:"serverless"`
	TLS        bool   `json:"tls"`
}

func ListCaches(ctx context.Context, cfg aws.Config) ([]Cache, error) {
	client := elasticache.NewFromConfig(cfg)
	var caches []Cache

	groups := elasticache.NewDescribeReplicationGroupsPaginator(client, &elasticache.DescribeReplicationGroupsInput{})
	for groups.HasMorePages() {
		page, err := groups.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("elasticache describe-replication-groups: %w", err)
		}
		for _, g := range page.ReplicationGroups {
			base := Cache{
				ClusterMode: aws.ToBool(g.ClusterEnabled),
				Engine:      aws.ToString(g.Engine),
				Name:        aws.ToString(g.ReplicationGroupId),
				TLS:         aws.ToBool(g.TransitEncryptionEnabled),
			}
			if g.ConfigurationEndpoint != nil {
				caches = append(caches, withEndpoint(base, g.ConfigurationEndpoint.Address, g.ConfigurationEndpoint.Port, "configuration"))
				continue
			}
			for _, ng := range g.NodeGroups {
				if ng.PrimaryEndpoint != nil {
					caches = append(caches, withEndpoint(base, ng.PrimaryEndpoint.Address, ng.PrimaryEndpoint.Port, "primary"))
				}
				if ng.ReaderEndpoint != nil {
					caches = append(caches, withEndpoint(base, ng.ReaderEndpoint.Address, ng.ReaderEndpoint.Port, "reader"))
				}
			}
		}
	}

	serverless := elasticache.NewDescribeServerlessCachesPaginator(client, &elasticache.DescribeServerlessCachesInput{})
	for serverless.HasMorePages() {
		page, err := serverless.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("elasticache describe-serverless-caches: %w", err)
		}
		for _, c := range page.ServerlessCaches {
			// Serverless caches always use TLS and expose one proxied endpoint, whatever the client mode.
			base := Cache{
				Engine:     aws.ToString(c.Engine),
				Name:       aws.ToString(c.ServerlessCacheName),
				Serverless: true,
				TLS:        true,
			}
			if c.Endpoint != nil {
				caches = append(caches, withEndpoint(base, c.Endpoint.Address, c.Endpoint.Port, "primary"))
			}
			if c.ReaderEndpoint != nil {
				caches = append(caches, withEndpoint(base, c.ReaderEndpoint.Address, c.ReaderEndpoint.Port, "reader"))
			}
		}
	}
	return caches, nil
}

func withEndpoint(c Cache, address *string, port *int32, role string) Cache {
	c.Host = aws.ToString(address)
	c.Port = int(aws.ToInt32(port))
	c.Role = role
	return c
}

// CacheIAMToken builds the password for an ElastiCache IAM-enabled user: a SigV4-presigned "connect" request
// for the cache, without its scheme. Valid 15 minutes, for new connections only.
func CacheIAMToken(ctx context.Context, cfg aws.Config, c Cache, user string) (string, error) {
	query := url.Values{"Action": {"connect"}, "User": {user}, "X-Amz-Expires": {"900"}}
	if c.Serverless {
		query.Set("ResourceType", "ServerlessCache")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+c.Name+"/?"+query.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("build connect request: %w", err)
	}

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return "", fmt.Errorf("retrieve credentials: %w", err)
	}
	signed, _, err := v4.NewSigner().PresignHTTP(ctx, creds, req, emptyPayloadSHA256, "elasticache", cfg.Region, time.Now())
	if err != nil {
		return "", fmt.Errorf("presign connect request: %w", err)
	}
	return strings.TrimPrefix(signed, "http://"), nil
}
