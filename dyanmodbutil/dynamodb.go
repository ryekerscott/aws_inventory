package dynamodbutil

import (
	"context"
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	ltypes "repo.com/path/inventory/types"
)

func getSize(size int64) string {
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

func processTags(objectTags []types.Tag) map[string]string {
	tags := map[string]string{"Name": "N/A", "CostCenter": "N/A", "Project": "N/A"}
	tagsToFind := []string{"Name", "CostCenter", "Project"}
	for _, tag := range objectTags {
		if slices.Contains(tagsToFind, string(*tag.Key)) {
			tags[string(*tag.Key)] = string(*tag.Value)
		}
	}
	return tags
}

func getTableData(data []types.TableDescription, dynamodb_client dynamodb.Client, account string, region string) [][]string {
	var tableArr [][]string
	for _, i := range data {
		tableTags, _ := dynamodb_client.ListTagsOfResource(context.TODO(), &dynamodb.ListTagsOfResourceInput{ResourceArn: i.TableArn})
		tags := processTags(tableTags.Tags)
		table := []string{
			account,
			region,
			aws.ToString(i.TableName),
			aws.ToString(i.TableId),
			getSize(aws.ToInt64(i.TableSizeBytes)),
			tags["CostCenter"],
			tags["Project"],
		}
		tableArr = append(tableArr, table)
	}
	return tableArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	tables := [][]string{
		{
			"Account",
			"Region",
			"Table Name",
			"Table ID",
			"Table Size",
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
		db_client := dynamodb.NewFromConfig(cfg)
		tableData := []types.TableDescription{}
		paginator := dynamodb.NewListTablesPaginator(db_client, &dynamodb.ListTablesInput{})
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.TableNames) == 0 {
				continue outer
			}
			for _, res := range page.TableNames {
				table, _ := db_client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: &res})
				tableData = append(tableData, *table.Table)
			}
		}
		tables = append(tables, getTableData(tableData, *db_client, name, region)...)
	}
	return tables

}
