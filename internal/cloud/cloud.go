package cloud

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/hashicorp/go-version"
	"github.com/rs/zerolog"
)

const (
	// DescribeDomains accepts at most 5 domain names per call.
	describeDomainsBatch = 5
	// DescribeInstanceInformation returns 10 items per page by default; 50 is the maximum.
	describeInstanceInformationPageSize = 50
)

// minMuxAgentVersion is the first SSM agent able to multiplex several TCP connections over one port-forward session.
var minMuxAgentVersion = version.Must(version.NewVersion("3.0.196.0"))

type Domain struct {
	Endpoint      string `json:"endpoint"`
	EngineVersion string `json:"engineVersion"`
	Name          string `json:"name"`
}

type Instance struct {
	AgentVersion string `json:"agentVersion"`
	ID           string `json:"id"`
	Name         string `json:"name"`
	Platform     string `json:"platform"`
}

// SupportsMux reports whether the instance agent can serve concurrent connections; older agents serve one at a time.
func (i Instance) SupportsMux() bool {
	v, err := version.NewVersion(i.AgentVersion)
	return err == nil && v.GreaterThanOrEqual(minMuxAgentVersion)
}

// AccountID returns the AWS account the credentials belong to. It doubles as a credentials check.
func AccountID(ctx context.Context, cfg aws.Config) (string, error) {
	out, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("sts get-caller-identity: %w", err)
	}
	return aws.ToString(out.Account), nil
}

func ListDomains(ctx context.Context, cfg aws.Config) ([]Domain, error) {
	client := opensearch.NewFromConfig(cfg)

	names, err := client.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
	if err != nil {
		return nil, fmt.Errorf("opensearch list-domain-names: %w", err)
	}

	domainNames := make([]string, 0, len(names.DomainNames))
	for _, d := range names.DomainNames {
		domainNames = append(domainNames, aws.ToString(d.DomainName))
	}

	domains := make([]Domain, 0, len(domainNames))
	for batch := range slices.Chunk(domainNames, describeDomainsBatch) {
		out, err := client.DescribeDomains(ctx, &opensearch.DescribeDomainsInput{DomainNames: batch})
		if err != nil {
			return nil, fmt.Errorf("opensearch describe-domains: %w", err)
		}
		for _, d := range out.DomainStatusList {
			endpoint := d.Endpoints["vpc"]
			if endpoint == "" {
				endpoint = aws.ToString(d.Endpoint)
			}
			domains = append(domains, Domain{
				Endpoint:      endpoint,
				EngineVersion: aws.ToString(d.EngineVersion),
				Name:          aws.ToString(d.DomainName),
			})
		}
	}
	return domains, nil
}

// ListInstances returns the managed nodes whose SSM agent is online, i.e. the ones a session can be opened to.
func ListInstances(ctx context.Context, cfg aws.Config) ([]Instance, error) {
	paginator := ssm.NewDescribeInstanceInformationPaginator(ssm.NewFromConfig(cfg), &ssm.DescribeInstanceInformationInput{
		Filters: []ssmtypes.InstanceInformationStringFilter{
			{Key: aws.String(string(ssmtypes.InstanceInformationFilterKeyPingStatus)), Values: []string{string(ssmtypes.PingStatusOnline)}},
		},
		MaxResults: aws.Int32(describeInstanceInformationPageSize),
	})

	var instances []Instance
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("ssm describe-instance-information: %w", err)
		}
		for _, i := range page.InstanceInformationList {
			instances = append(instances, Instance{
				AgentVersion: aws.ToString(i.AgentVersion),
				ID:           aws.ToString(i.InstanceId),
				// Hybrid nodes carry their activation name; EC2 ones get the Name tag below.
				Name:     cmp.Or(aws.ToString(i.Name), aws.ToString(i.ComputerName)),
				Platform: cmp.Or(aws.ToString(i.PlatformName), string(i.PlatformType)),
			})
		}
	}

	// Names are cosmetic: without ec2:DescribeTags the picker still works, showing IDs only.
	names, err := instanceNames(ctx, cfg)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("could not read instance Name tags")
	}
	for idx := range instances {
		if name, ok := names[instances[idx].ID]; ok {
			instances[idx].Name = name
		}
	}
	return instances, nil
}

// instanceNames maps EC2 instance IDs to their Name tag.
func instanceNames(ctx context.Context, cfg aws.Config) (map[string]string, error) {
	paginator := ec2.NewDescribeTagsPaginator(ec2.NewFromConfig(cfg), &ec2.DescribeTagsInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("resource-type"), Values: []string{"instance"}},
			{Name: aws.String("key"), Values: []string{"Name"}},
		},
	})

	names := map[string]string{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("ec2 describe-tags: %w", err)
		}
		for _, t := range page.Tags {
			names[aws.ToString(t.ResourceId)] = aws.ToString(t.Value)
		}
	}
	return names, nil
}
