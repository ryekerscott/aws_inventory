package emrutil

import (
	"context"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/aws/aws-sdk-go-v2/service/emr/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getClusterTags(cluster types.Cluster) map[string]string {
	tags := map[string]string{"CostCenter": "N/A", "Project": "N/A"}
	tagsToFind := []string{"CostCenter", "Project"}
	clusterTags := cluster.Tags
	for _, tag := range clusterTags {
		if slices.Contains(tagsToFind, string(*tag.Key)) {
			tags[string(*tag.Key)] = string(*tag.Value)
		}
	}
	return tags
}

func getClusterData(data []types.Cluster, emr_client emr.Client, account string, region string) [][]string {
	var clusterArr [][]string
	for _, i := range data {
		tags := getClusterTags(i)
		cluster := []string{
			account,
			region,
			aws.ToString(i.Name),
			aws.ToString(i.Id),
			string(i.Status.State),
			aws.ToString(i.CustomAmiId),
			aws.ToString(i.ReleaseLabel),
			tags["CostCenter"],
			tags["Project"],
		}
		clusterArr = append(clusterArr, cluster)
	}
	return clusterArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	clusters := [][]string{
		{
			"Account",
			"Region",
			"Name",
			"ID",
			"State",
			"AMI",
			"Release",
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
		emr_client := emr.NewFromConfig(cfg)
		clo, _ := emr_client.ListClusters(
			ctx,
			&emr.ListClustersInput{ClusterStates: []types.ClusterState{types.ClusterStateStarting, types.ClusterStateBootstrapping, types.ClusterStateRunning, types.ClusterStateWaiting}},
		)
		emrClusters := []types.Cluster{}
		for _, x := range clo.Clusters {
			cluster, _ := emr_client.DescribeCluster(
				ctx,
				&emr.DescribeClusterInput{
					ClusterId: x.Id,
				},
			)
			emrClusters = append(emrClusters, *cluster.Cluster)
		}
		if len(emrClusters) != 0 {
			clusters = append(clusters, getClusterData(emrClusters, *emr_client, name, region)...)
		}
	}
	return clusters
}
