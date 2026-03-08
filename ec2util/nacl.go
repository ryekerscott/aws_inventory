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

func getNaclData(data []types.NetworkAcl, ec2_client ec2.Client, account string, region string) [][]string {
	var naclArr [][]string
	for _, i := range data {
		tags := processTags(i.Tags)
		nacl := []string{
			account,
			region,
			aws.ToString(i.NetworkAclId),
			tags["Name"],
			aws.ToString(i.VpcId),
			vpcInfo(ec2_client, aws.ToString(i.VpcId)),
			tags["CostCenter"],
			tags["Project"],
		}
		naclArr = append(naclArr, nacl)
	}
	return naclArr
}

func Inventory_Nacl(credMap map[string]ltypes.Env) [][]string {
	nacls := [][]string{
		{
			"Account",
			"Region",
			"NACL ID",
			"NACL Name",
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
		naclData := []types.NetworkAcl{}
		paginator := ec2.NewDescribeNetworkAclsPaginator(ec2_client, &ec2.DescribeNetworkAclsInput{})
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.NetworkAcls) == 0 {
				continue outer
			}
			for _, res := range page.NetworkAcls {
				naclData = append(naclData, res)
			}
		}
		nacls = append(nacls, getNaclData(naclData, *ec2_client, name, region)...)
	}
	return nacls

}
