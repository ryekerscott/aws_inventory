package ec2util

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getRtbData(data []types.RouteTable, ec2_client ec2.Client, account string, region string) [][]string {
	var rtbArr [][]string
	for _, i := range data {
		tags := processTags(i.Tags)
		rtb := []string{
			account,
			region,
			aws.ToString(i.RouteTableId),
			tags["Name"],
			aws.ToString(i.VpcId),
			vpcInfo(ec2_client, aws.ToString(i.VpcId)),
			tags["CostCenter"],
			tags["Project"],
		}
		rtbArr = append(rtbArr, rtb)
	}
	return rtbArr
}

func Inventory_RTB(credMap map[string]ltypes.Env) [][]string {
	routeTables := [][]string{
		{
			"Account",
			"Region",
			"RTB ID",
			"RTB Name",
			"VPC ID",
			"VPC Name",
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
		ec2_client := ec2.NewFromConfig(cfg)
		rtbData := []types.RouteTable{}
		paginator := ec2.NewDescribeRouteTablesPaginator(ec2_client, &ec2.DescribeRouteTablesInput{})
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.RouteTables) == 0 {
				continue outer
			}
			for _, res := range page.RouteTables {
				rtbData = append(rtbData, res)
			}
		}
		routeTables = append(routeTables, getRtbData(rtbData, *ec2_client, name, region)...)
	}
	return routeTables

}
