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

func getSgData(data []types.SecurityGroup, ec2_client ec2.Client, account string, region string) [][]string {
	var sgArr [][]string
	for _, i := range data {
		tags := processTags(i.Tags)
		sg := []string{
			account,
			region,
			aws.ToString(i.SecurityGroupArn),
			aws.ToString(i.GroupName),
			aws.ToString(i.VpcId),
			vpcInfo(ec2_client, aws.ToString(i.VpcId)),
			tags["CostCenter"],
			tags["Project"],
		}
		sgArr = append(sgArr, sg)
	}
	return sgArr
}

func Inventory_Sg(credMap map[string]ltypes.Env) [][]string {
	sgs := [][]string{
		{
			"Account",
			"Region",
			"SG ARN",
			"SG Name",
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
		sgData := []types.SecurityGroup{}
		paginator := ec2.NewDescribeSecurityGroupsPaginator(ec2_client, &ec2.DescribeSecurityGroupsInput{})
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.SecurityGroups) == 0 {
				continue outer
			}
			for _, res := range page.SecurityGroups {
				sgData = append(sgData, res)
			}
		}
		sgs = append(sgs, getSgData(sgData, *ec2_client, name, region)...)
	}
	return sgs

}
