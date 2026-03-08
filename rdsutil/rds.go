package rdsutil

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func processTags(instance rdstypes.DBInstance) map[string]string {
	tags := map[string]string{"Name": "N/A", "CostCenter": "N/A", "Project": "N/A", "LifeCycle": "N/A"}
	tagsToFind := []string{"Name", "CostCenter", "Project", "LifeCycle"}
	instanceTags := instance.TagList
	for _, tag := range instanceTags {
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
	vpcTags := vpcData.Vpcs[0].Tags
	vpcName := "N/A"
	for _, tag := range vpcTags {
		if *tag.Key == "Name" {
			vpcName = string(*tag.Value)
		}
	}
	return vpcName
}

func getInstanceData(data []rdstypes.DBInstance, ec2_client ec2.Client, account string) [][]string {
	instanceArr := [][]string{}
	for _, i := range data {
		ec2TypeStr := strings.TrimPrefix(aws.ToString(i.DBInstanceClass), "db.")
		ec2Type := ec2types.InstanceType(ec2TypeStr)
		instType, _ := ec2_client.DescribeInstanceTypes(
			context.TODO(),
			&ec2.DescribeInstanceTypesInput{
				InstanceTypes: []ec2types.InstanceType{ec2Type},
			},
		)
		tags := processTags(i)
		vpcId := aws.ToString(i.DBSubnetGroup.VpcId)
		vpcName := vpcInfo(ec2_client, vpcId)
		instance := []string{
			account,
			tags["Name"],
			aws.ToString(i.DBInstanceIdentifier),
			aws.ToString(i.Endpoint.Address),
			fmt.Sprintf("%d", aws.ToInt32(i.Endpoint.Port)),
			aws.ToString(i.AvailabilityZone),
			aws.ToString(i.Engine),
			aws.ToString(i.EngineVersion),
			aws.ToString(i.StorageType),
			aws.ToString(i.DBInstanceClass),
			fmt.Sprintf("%d", aws.ToInt32(instType.InstanceTypes[0].VCpuInfo.DefaultVCpus)),
			fmt.Sprintf("%dGB", aws.ToInt64(instType.InstanceTypes[0].MemoryInfo.SizeInMiB)/1024),
			fmt.Sprintf("%dGB", aws.ToInt32(i.AllocatedStorage)),
			vpcId,
			vpcName,
			tags["LifeCycle"],
			tags["CostCenter"],
			tags["Project"],
		}
		instanceArr = append(instanceArr, instance)
	}
	return instanceArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	instances := [][]string{
		{
			"Account",
			"Name",
			"Identifier",
			"Endpoint",
			"Port",
			"Availablility Zone",
			"Engine",
			"Engine Version",
			"Storage Type",
			"Instance Type",
			"CPU",
			"Memory",
			"Allocated Storage",
			"VPC ID",
			"VPC Name",
			"Environment",
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
		rds_client := rds.NewFromConfig(cfg)
		ec2_client := ec2.NewFromConfig(cfg)
		instanceData := []types.DBInstance{}
		paginator := rds.NewDescribeDBInstancesPaginator(
			rds_client,
			&rds.DescribeDBInstancesInput{},
		)
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.DBInstances) == 0 {
				continue outer
			}
			for _, dbInstance := range page.DBInstances {
				instanceData = append(instanceData, dbInstance)
			}
		}
		instances = append(instances, getInstanceData(instanceData, *ec2_client, name)...)
	}
	return instances
}
