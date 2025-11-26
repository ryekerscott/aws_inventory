package eksutil

import (
	"context"

	ltypes "repo.com/path/inventory/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

func getEksData(data []types.Cluster, eks_client eks.Client, account string, region string) [][]string {
	var eksArr [][]string
	for _, i := range data {
		costCenter := "N/A"
		project := "N/A"
		if i.Tags["CostCenter"] != "" {
			costCenter = i.Tags["CostCenter"]
		}
		if i.Tags["Project"] != "" {
			project = i.Tags["Project"]
		}
		cluster := []string{
			account,
			region,
			aws.ToString(i.Name),
			aws.ToString(i.Version),
			aws.ToString(i.Arn),
			costCenter,
			project,
		}
		eksArr = append(eksArr, cluster)
	}
	return eksArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	clusters := [][]string{
		{
			"Account",
			"Region",
			"Name",
			"Version",
			"Arn",
			"CostCenter",
			"Project",
		},
	}
outer:
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
		eks_client := eks.NewFromConfig(cfg)
		eksData := []types.Cluster{}
		paginator := eks.NewListClustersPaginator(
			eks_client,
			&eks.ListClustersInput{},
		)
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Clusters) == 0 {
				continue outer
			}
			for _, res := range page.Clusters {
				cluster, _ := eks_client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: &res})
				eksData = append(eksData, *cluster.Cluster)
			}
		}
		clusters = append(clusters, getEksData(eksData, *eks_client, name, region)...)
	}
	return clusters

}
