package cloud

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

// Database is an RDS endpoint a client can connect to: a cluster writer/reader endpoint or a standalone instance.
type Database struct {
	Engine        string `json:"engine"`
	EngineVersion string `json:"engineVersion"`
	Host          string `json:"host"`
	IAMAuth       bool   `json:"iamAuth"`
	MasterUser    string `json:"masterUser"`
	Name          string `json:"name"`
	Port          int    `json:"port"`
	Role          string `json:"role"`
}

// ListDatabases returns Aurora/Multi-AZ cluster endpoints (writer and reader) and instances not part of a cluster.
func ListDatabases(ctx context.Context, cfg aws.Config) ([]Database, error) {
	client := rds.NewFromConfig(cfg)
	var databases []Database

	clusters := rds.NewDescribeDBClustersPaginator(client, &rds.DescribeDBClustersInput{})
	for clusters.HasMorePages() {
		page, err := clusters.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("rds describe-db-clusters: %w", err)
		}
		for _, c := range page.DBClusters {
			base := Database{
				Engine:        aws.ToString(c.Engine),
				EngineVersion: aws.ToString(c.EngineVersion),
				IAMAuth:       aws.ToBool(c.IAMDatabaseAuthenticationEnabled),
				MasterUser:    aws.ToString(c.MasterUsername),
				Name:          aws.ToString(c.DBClusterIdentifier),
				Port:          int(aws.ToInt32(c.Port)),
			}
			if c.Endpoint != nil {
				writer := base
				writer.Host = aws.ToString(c.Endpoint)
				writer.Role = "writer"
				databases = append(databases, writer)
			}
			if c.ReaderEndpoint != nil {
				reader := base
				reader.Host = aws.ToString(c.ReaderEndpoint)
				reader.Role = "reader"
				databases = append(databases, reader)
			}
		}
	}

	instances := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{})
	for instances.HasMorePages() {
		page, err := instances.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("rds describe-db-instances: %w", err)
		}
		for _, i := range page.DBInstances {
			// Cluster members are reached through the cluster endpoints; instances still creating have no endpoint.
			if i.DBClusterIdentifier != nil || i.Endpoint == nil {
				continue
			}
			databases = append(databases, Database{
				Engine:        aws.ToString(i.Engine),
				EngineVersion: aws.ToString(i.EngineVersion),
				Host:          aws.ToString(i.Endpoint.Address),
				IAMAuth:       aws.ToBool(i.IAMDatabaseAuthenticationEnabled),
				MasterUser:    aws.ToString(i.MasterUsername),
				Name:          aws.ToString(i.DBInstanceIdentifier),
				Port:          int(aws.ToInt32(i.Endpoint.Port)),
				Role:          "instance",
			})
		}
	}
	return databases, nil
}
