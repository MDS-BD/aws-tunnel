package cloud

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

type Cluster struct {
	// CAData is the PEM cluster CA, decoded from the base64 EKS returns.
	CAData        []byte `json:"caData"`
	Endpoint      string `json:"endpoint"`
	Name          string `json:"name"`
	PrivateAccess bool   `json:"privateAccess"`
	PublicAccess  bool   `json:"publicAccess"`
	Version       string `json:"version"`
}

// Host is the API server host name, what its certificate is issued for.
func (c Cluster) Host() (string, error) {
	u, err := url.Parse(c.Endpoint)
	if err != nil || u.Hostname() == "" {
		return "", fmt.Errorf("cluster %s has no valid endpoint %q", c.Name, c.Endpoint)
	}
	return u.Hostname(), nil
}

// ListClusters returns clusters reachable from the VPC, i.e. with private endpoint access.
func ListClusters(ctx context.Context, cfg aws.Config) ([]Cluster, error) {
	client := eks.NewFromConfig(cfg)

	var clusters []Cluster
	paginator := eks.NewListClustersPaginator(client, &eks.ListClustersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("eks list-clusters: %w", err)
		}
		for _, name := range page.Clusters {
			c, err := DescribeCluster(ctx, cfg, name)
			if err != nil {
				return nil, err
			}
			if c.PrivateAccess {
				clusters = append(clusters, c)
			}
		}
	}
	return clusters, nil
}

func DescribeCluster(ctx context.Context, cfg aws.Config, name string) (Cluster, error) {
	out, err := eks.NewFromConfig(cfg).DescribeCluster(ctx, &eks.DescribeClusterInput{Name: aws.String(name)})
	if err != nil {
		return Cluster{}, fmt.Errorf("eks describe-cluster %s: %w", name, err)
	}

	c := out.Cluster
	cluster := Cluster{
		Endpoint: aws.ToString(c.Endpoint),
		Name:     aws.ToString(c.Name),
		Version:  aws.ToString(c.Version),
	}
	if c.ResourcesVpcConfig != nil {
		cluster.PrivateAccess = c.ResourcesVpcConfig.EndpointPrivateAccess
		cluster.PublicAccess = c.ResourcesVpcConfig.EndpointPublicAccess
	}
	if c.CertificateAuthority != nil {
		ca, err := base64.StdEncoding.DecodeString(aws.ToString(c.CertificateAuthority.Data))
		if err != nil {
			return Cluster{}, fmt.Errorf("decode %s certificate authority: %w", name, err)
		}
		cluster.CAData = ca
	}
	return cluster, nil
}

// ClusterToken returns a bearer token for the cluster API server, the same one `aws eks get-token` prints.
func ClusterToken(ctx context.Context, cfg aws.Config, clusterName string) (token.Token, error) {
	gen, err := token.NewGenerator(false, false)
	if err != nil {
		return token.Token{}, fmt.Errorf("token generator: %w", err)
	}
	t, err := gen.GetWithSTS(clusterName, sts.NewFromConfig(cfg))
	if err != nil {
		return token.Token{}, fmt.Errorf("presign sts get-caller-identity: %w", err)
	}
	return t, nil
}
