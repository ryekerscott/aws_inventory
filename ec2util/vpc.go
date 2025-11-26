package ec2util

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ltypes "repo.com/path/inventory/types"
)

func getVpcData(data []types.Vpc, ec2_client ec2.Client, account string, region string) [][]string {
	var vpcArr [][]string
	for _, i := range data {
		tags := processTags(i.Tags)
		vpc := []string{
			account,
			region,
			tags["Name"],
			aws.ToString(i.VpcId),
			aws.ToString(i.CidrBlock),
			tags["CostCenter"],
			tags["Project"],
		}
		vpcArr = append(vpcArr, vpc)
	}
	return vpcArr
}

func Inventory_VPC(credMap map[string]ltypes.Env) [][]string {
	Vpcs := [][]string{
		{
			"Account",
			"Region",
			"VPC Name",
			"VPC ID",
			"CIDR",
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
		vpcData := []types.Vpc{}
		paginator := ec2.NewDescribeVpcsPaginator(ec2_client, &ec2.DescribeVpcsInput{})
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Vpcs) == 0 {
				continue outer
			}
			for _, res := range page.Vpcs {
				vpcData = append(vpcData, res)
			}
		}
		Vpcs = append(Vpcs, getVpcData(vpcData, *ec2_client, name, region)...)
	}
	return Vpcs

}
