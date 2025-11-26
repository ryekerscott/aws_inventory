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

func getVolumeData(data []types.Volume, ebs_client ec2.Client, account string, region string) [][]string {
	var ebsArr [][]string
	for _, i := range data {
		instanceId := "N/A"
		instanceName := "N/A"
		if string(i.State) == "in-use" {
			instanceId = aws.ToString(i.Attachments[0].InstanceId)
			instance, _ := ebs_client.DescribeInstances(
				context.TODO(),
				&ec2.DescribeInstancesInput{
					InstanceIds: []string{instanceId},
				},
			)
			tags := instance.Reservations[0].Instances[0].Tags
			for _, v := range tags {
				if aws.ToString(v.Key) == "Name" {
					instanceName = aws.ToString(v.Value)
				}
			}
		}
		tags := processTags(i.Tags)
		size := aws.ToInt32(i.Size)
		sizeStr := "GB"
		if size >= 1024 {
			size = size / 1024
			sizeStr = "TB"
		}

		volume := []string{
			account,
			aws.ToString(i.VolumeId),
			aws.ToString(i.AvailabilityZone),
			string(i.State),
			fmt.Sprintf("%t", *i.Encrypted),
			string(i.VolumeType),
			fmt.Sprintf("%d %s", size, sizeStr),
			instanceId,
			instanceName,
			tags["CostCenter"],
			tags["Project"],
		}
		ebsArr = append(ebsArr, volume)
	}
	return ebsArr
}

func Inventory_EBS(credMap map[string]ltypes.Env) [][]string {
	volumes := [][]string{
		{
			"Account",
			"Volume ID",
			"Availability Zone",
			"State",
			"Encrypted",
			"Volume Type",
			"Volume Capacity",
			"Instance ID",
			"Instance Name",
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
		ebs_client := ec2.NewFromConfig(cfg)
		ebsData := []types.Volume{}
		paginator := ec2.NewDescribeVolumesPaginator(
			ebs_client,
			&ec2.DescribeVolumesInput{},
		)
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Volumes) == 0 {
				continue outer
			}
			for _, res := range page.Volumes {
				ebsData = append(ebsData, res)
			}
		}
		volumes = append(volumes, getVolumeData(ebsData, *ebs_client, name, region)...)
	}
	return volumes

}
