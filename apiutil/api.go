package apiutil

import (
	"context"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getApiData(data []types.RestApi, account string, region string) [][]string {
	var apiArr [][]string

	for _, i := range data {
		endpointType := make([]string, len(i.EndpointConfiguration.Types))
		for a, x := range i.EndpointConfiguration.Types {
			endpointType[a] = string(x)
		}
		tags := map[string]string{"CostCenter": "N/A", "Project": "N/A"}
		tagValsToFind := []string{"CostCenter", "Project"}
		for k, v := range i.Tags {
			if slices.Contains(tagValsToFind, k) {
				tags[k] = v
			}
		}
		api := []string{
			account,
			region,
			aws.ToString(i.Name),
			aws.ToString(i.Id),
			strings.Join(i.EndpointConfiguration.VpcEndpointIds, ","),
			strings.Join(endpointType, ","),
			aws.ToString(i.Description),
			tags["CostCenter"],
			tags["Project"],
		}
		apiArr = append(apiArr, api)
	}
	return apiArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	apis := [][]string{
		{
			"Account",
			"Region",
			"Api Name",
			"API ID",
			"VPC Endpoints",
			"Endpoint Types",
			"API Description",
			"CostCenter",
			"Project",
		},
	}
	for _, v := range credMap {
		creds := v.Credentials
		if v.Credentials["access"] == "" {
			panic("No Credentials Found")
		}
		access := creds["access"]
		secret := creds["secret"]
		region := creds["region"]
		name := v.Name
		ctx := context.TODO()
		provider := credentials.NewStaticCredentialsProvider(access, secret, "")
		cfg, _ := config.LoadDefaultConfig(
			ctx,
			config.WithCredentialsProvider(provider),
			config.WithRegion(region),
		)
		api_client := apigateway.NewFromConfig(cfg)
		apiData := []types.RestApi{}
		var position *string
		for {
			page, _ := api_client.GetRestApis(
				ctx,
				&apigateway.GetRestApisInput{Position: position},
			)
			if len(page.Items) != 0 {
				apiData = append(apiData, page.Items...)
			}
			if page.Position != nil {
				position = page.Position
			} else {
				break
			}
		}
		apis = append(apis, getApiData(apiData, name, region)...)
	}

	return apis
}
