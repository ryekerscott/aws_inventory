package cloudwatchutil

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	ltypes "repo.com/path/inventory/types"
)

func getLgSize(size int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
		TB = 1 << 40
	)
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

func getLgData(data []types.LogGroup, cw_client cloudwatchlogs.Client, account string, region string) [][]string {
	var lgArr [][]string
	for _, i := range data {
		logGroup := []string{
			account,
			region,
			aws.ToString(i.LogGroupName),
			string(i.LogGroupClass),
			fmt.Sprintf("%d", aws.ToInt32(i.RetentionInDays)),
			getLgSize(*i.StoredBytes),
		}
		lgArr = append(lgArr, logGroup)
	}
	return lgArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	logGroups := [][]string{
		{
			"Account",
			"Region",
			"Log Group Name",
			"Log Group Class",
			"Log Retention (Days)",
			"Size",
		},
	}
outer:
	for _, v := range credMap {
		creds := v.Credentials
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
		cw_client := cloudwatchlogs.NewFromConfig(cfg)
		lgData := []types.LogGroup{}
		paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(
			cw_client,
			&cloudwatchlogs.DescribeLogGroupsInput{},
		)
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.LogGroups) == 0 {
				continue outer
			}
			for _, lg := range page.LogGroups {
				lgData = append(lgData, lg)
			}
		}
		logGroups = append(logGroups, getLgData(lgData, *cw_client, name, region)...)
	}
	return logGroups
}
