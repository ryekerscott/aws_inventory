package ec2util

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ltypes "repo.com/path/inventory/types"
)

func amiInfo(client ec2.Client, amiId string) map[string]string {
	amiData, _ := client.DescribeImages(
		context.TODO(),
		&ec2.DescribeImagesInput{
			ImageIds: []string{amiId},
		},
	)
	amiInfo := map[string]string{"Name": "N/A", "Operating System": "N/A", "Architecture": "N/A"}
	if len(amiData.Images) > 0 {
		amiInfo["Name"] = *amiData.Images[0].Name
		amiInfo["Operating System"] = *amiData.Images[0].PlatformDetails
		amiInfo["Architecture"] = string(amiData.Images[0].Architecture)
	}
	return amiInfo
}

func getInstanceData(data []types.Instance, ec2_client ec2.Client, account string) [][]string {
	var instanceArr [][]string
	for _, i := range data {
		if string(i.State.Name) == "terminated" {
			continue
		}
		instType, _ := ec2_client.DescribeInstanceTypes(
			context.TODO(),
			&ec2.DescribeInstanceTypesInput{
				InstanceTypes: []types.InstanceType{i.InstanceType},
			},
		)
		tags := processTags(i.Tags)
		ami := aws.ToString(i.ImageId)
		amiInfo := amiInfo(ec2_client, ami)
		vpcId, subnetId := aws.ToString(i.VpcId), aws.ToString(i.SubnetId)
		vpcName, subnetName := vpcInfo(ec2_client, vpcId), subnetInfo(ec2_client, subnetId)
		instance := []string{
			account,
			tags["Name"],
			aws.ToString(i.InstanceId),
			aws.ToString(i.PrivateIpAddress),
			aws.ToString(i.Placement.AvailabilityZone),
			ami,
			amiInfo["Name"],
			amiInfo["Architecture"],
			string(i.InstanceType),
			fmt.Sprintf("%d", aws.ToInt32(instType.InstanceTypes[0].VCpuInfo.DefaultVCpus)),
			fmt.Sprintf("%dGB", aws.ToInt64(instType.InstanceTypes[0].MemoryInfo.SizeInMiB)/1024),
			amiInfo["Operating System"],
			string(i.State.Name),
			vpcId,
			vpcName,
			subnetId,
			subnetName,
			tags["CostCenter"],
			tags["Project"],
		}
		instanceArr = append(instanceArr, instance)
	}
	return instanceArr
}

func Inventory_EC2(credMap map[string]ltypes.Env) [][]string {
	instances := [][]string{
		{
			"Account",
			"Name",
			"Instance ID",
			"IP Address",
			"Availability Zone",
			"AMI ID",
			"AMI Name",
			"Architecture",
			"Instance Type",
			"CPU",
			"Memory",
			"Operating System",
			"Instance State",
			"VPC ID",
			"VPC Name",
			"Subnet ID",
			"Subnet Name",
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
		instanceData := []types.Instance{}
		paginator := ec2.NewDescribeInstancesPaginator(ec2_client, &ec2.DescribeInstancesInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				panic(err.Error())
			}
			if len(page.Reservations) == 0 {
				continue outer
			}
			for _, res := range page.Reservations {
				instanceData = append(instanceData, res.Instances...)
			}
		}
		instances = append(instances, getInstanceData(instanceData, *ec2_client, name)...)
	}
	return instances

}
