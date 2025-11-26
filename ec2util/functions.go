package ec2util

import (
	"context"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func processTags(objectTags []types.Tag) map[string]string {
	tags := map[string]string{"Name": "N/A", "CostCenter": "N/A", "Project": "N/A"}
	tagsToFind := []string{"Name", "CostCenter", "Project"}
	for _, tag := range objectTags {
		if slices.Contains(tagsToFind, string(*tag.Key)) {
			tags[string(*tag.Key)] = string(*tag.Value)
		}
	}
	return tags
}

func vpcInfo(client ec2.Client, vpcId string) string {
	vpcData, err := client.DescribeVpcs(
		context.TODO(),
		&ec2.DescribeVpcsInput{
			VpcIds: []string{vpcId},
		},
	)
	if err != nil {
		panic(err.Error())
	}
	time.Sleep(100 * time.Millisecond)
	vpcTags := vpcData.Vpcs[0].Tags
	vpcName := "N/A"
	for _, tag := range vpcTags {
		if *tag.Key == "Name" {
			vpcName = string(*tag.Value)
		}
	}
	return vpcName
}

func subnetInfo(client ec2.Client, subnetId string) string {
	subnetData, err := client.DescribeSubnets(
		context.TODO(),
		&ec2.DescribeSubnetsInput{
			SubnetIds: []string{subnetId},
		},
	)
	if err != nil {
		panic(err.Error())
	}
	subnetTags := subnetData.Subnets[0].Tags
	subnetName := "N/A"
	for _, tag := range subnetTags {
		if *tag.Key == "Name" {
			subnetName = string(*tag.Value)
		}
	}
	return subnetName
}
