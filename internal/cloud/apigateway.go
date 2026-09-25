package cloud

import (
	"context"
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	apigwtypes "github.com/aws/aws-sdk-go-v2/service/apigateway/types"
)

// API is a stage of a private REST API. Private APIs only answer through an execute-api VPC endpoint.
type API struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Stage string `json:"stage"`
}

// Host is the private invoke host name. It resolves to the VPC endpoint when private DNS is enabled on it.
func (a API) Host(region string) string {
	return a.ID + ".execute-api." + region + ".amazonaws.com"
}

// ListPrivateAPIs returns one entry per stage of every private REST API.
func ListPrivateAPIs(ctx context.Context, cfg aws.Config) ([]API, error) {
	client := apigateway.NewFromConfig(cfg)

	var apis []API
	paginator := apigateway.NewGetRestApisPaginator(client, &apigateway.GetRestApisInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("apigateway get-rest-apis: %w", err)
		}
		for _, api := range page.Items {
			if api.EndpointConfiguration == nil ||
				!slices.Contains(api.EndpointConfiguration.Types, apigwtypes.EndpointTypePrivate) {
				continue
			}
			stages, err := client.GetStages(ctx, &apigateway.GetStagesInput{RestApiId: api.Id})
			if err != nil {
				return nil, fmt.Errorf("apigateway get-stages %s: %w", aws.ToString(api.Id), err)
			}
			for _, s := range stages.Item {
				apis = append(apis, API{
					ID:    aws.ToString(api.Id),
					Name:  aws.ToString(api.Name),
					Stage: aws.ToString(s.StageName),
				})
			}
		}
	}
	return apis, nil
}
