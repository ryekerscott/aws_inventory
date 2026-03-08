package efsutil

import (
	"context"
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	efstypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getEfsTags(fstags []efstypes.Tag) map[string]string {
	tags := map[string]string{"CostCenter": "N/A", "Project": "N/A"}
	tagsToFind := []string{"CostCenter", "Project"}
	for _, tag := range fstags {
		if slices.Contains(tagsToFind, string(*tag.Key)) {
			tags[string(*tag.Key)] = string(*tag.Value)
		}
	}
	return tags
}

func getSize(fsize efstypes.FileSystemSize) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
		TB = 1 << 40
	)
	size := fsize.Value
	switch {
	case size >= TB:
		return fmt.Sprintf("%d TB", size/TB)
	case size >= GB:
		return fmt.Sprintf("%d GB", size/GB)
	case size >= MB:
		return fmt.Sprintf("%d MB", size/MB)
	case size >= KB:
		return fmt.Sprintf("%d KB", size/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func getEfsData(data []efstypes.FileSystemDescription, efs_client efs.Client, account string, region string) [][]string {
	var efsArr [][]string

	for _, i := range data {
		tags := getEfsTags(i.Tags)
		fs := []string{
			account,
			aws.ToString(i.Name),
			region,
			fmt.Sprintf("%t", *i.Encrypted),
			getSize(*i.SizeInBytes),
			tags["CostCenter"],
			tags["Project"],
		}
		efsArr = append(efsArr, fs)
	}
	return efsArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	filesystems := [][]string{
		{
			"Account",
			"Name",
			"region",
			"Encryption Status",
			"Size",
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
		efs_client := efs.NewFromConfig(cfg)
		efsData := []efstypes.FileSystemDescription{}
		paginator := efs.NewDescribeFileSystemsPaginator(
			efs_client,
			&efs.DescribeFileSystemsInput{},
		)
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.FileSystems) == 0 {
				continue outer
			}
			for _, res := range page.FileSystems {
				efsData = append(efsData, res)
			}
		}
		filesystems = append(filesystems, getEfsData(efsData, *efs_client, name, region)...)
	}
	return filesystems

}
