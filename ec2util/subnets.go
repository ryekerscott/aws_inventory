package ec2util

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getSubnetData(data []types.Subnet, ec2_client ec2.Client, account string, region string) [][]string {
	var subnetArr [][]string
	for _, i := range data {
		tags := processTags(i.Tags)
		subnet := []string{
			account,
			region,
			tags["Name"],
			aws.ToString(i.SubnetId),
			aws.ToString(i.CidrBlock),
			aws.ToString(i.AvailabilityZone),
			fmt.Sprintf("%d", aws.ToInt32(i.AvailableIpAddressCount)),
			aws.ToString(i.VpcId),
			vpcInfo(ec2_client, aws.ToString(i.VpcId)),
			tags["CostCenter"],
			tags["Project"],
		}
		subnetArr = append(subnetArr, subnet)
	}
	return subnetArr
}

func Inventory_Subnets(credMap map[string]ltypes.Env) [][]string {
	subnets := [][]string{
		{
			"Account",
			"Region",
			"Subnet Name",
			"Subnet ID",
			"CIDR",
			"Availability Zone",
			"Available IP Addresses",
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
		subnetData := []types.Subnet{}
		paginator := ec2.NewDescribeSubnetsPaginator(ec2_client, &ec2.DescribeSubnetsInput{})
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Subnets) == 0 {
				continue outer
			}
			for _, res := range page.Subnets {
				subnetData = append(subnetData, res)
			}
		}
		subnets = append(subnets, getSubnetData(subnetData, *ec2_client, name, region)...)
	}
	return subnets

}
