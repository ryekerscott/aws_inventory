package s3util

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getBucketSize(bucket string, cw_client cloudwatch.Client) string {
	end := time.Now().Truncate(24 * time.Hour).Add(-24 * time.Hour)
	start := end.Add(-24 * time.Hour)
	data, _ := cw_client.GetMetricStatistics(
		context.TODO(),
		&cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/S3"),
			MetricName: aws.String("BucketSizeBytes"),
			Dimensions: []cwTypes.Dimension{
				{
					Name:  aws.String("BucketName"),
					Value: &bucket,
				},
				{
					Name:  aws.String("StorageType"),
					Value: aws.String("StandardStorage"),
				},
			},
			StartTime: &start,
			EndTime:   &end,
			Period:    aws.Int32(int32(86400)),
			Statistics: []cwTypes.Statistic{
				cwTypes.StatisticAverage,
			},
		},
	)
	size := 0.0
	if len(*&data.Datapoints) != 0 {
		size = *data.Datapoints[0].Average
	}

	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
		TB = 1 << 40
	)
	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", size/TB)
	case size >= GB:
		return fmt.Sprintf("%.2f GB", size/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", size/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", size/KB)
	default:
		return fmt.Sprintf("%.2f B", size)
	}

}

func getBucketTags(bucket types.Bucket, s3_client s3.Client) map[string]string {
	tagsToFind := []string{"CostCenter", "Project"}
	tags := map[string]string{
		"CostCenter": "N/A",
		"Project":    "N/A",
	}
	bucketTags, err := s3_client.GetBucketTagging(
		context.TODO(),
		&s3.GetBucketTaggingInput{
			Bucket: bucket.Name,
		},
	)
	if err != nil {
		return tags
	}

	for _, v := range bucketTags.TagSet {
		if slices.Contains(tagsToFind, aws.ToString(v.Key)) {
			tags[aws.ToString(v.Key)] = aws.ToString(v.Value)
		}
	}

	return tags
}

func getBucketData(data []types.Bucket, s3_client s3.Client, cw_client cloudwatch.Client, account string, region string) [][]string {
	var bucketArr [][]string
	for _, i := range data {
		bucketRegion, _ := s3_client.GetBucketLocation(context.TODO(), &s3.GetBucketLocationInput{Bucket: i.Name})

		if string(bucketRegion.LocationConstraint) != region {
			continue
		}
		tags := getBucketTags(i, s3_client)
		bucket := []string{
			account,
			aws.ToString(i.Name),
			string(bucketRegion.LocationConstraint),
			getBucketSize(aws.ToString(i.Name), cw_client),
			tags["CostCenter"],
			tags["Project"],
		}
		bucketArr = append(bucketArr, bucket)
	}
	return bucketArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	buckets := [][]string{
		{
			"Account",
			"Bucket Name",
			"Region",
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
		s3_client := s3.NewFromConfig(cfg)
		cw_client := cloudwatch.NewFromConfig(cfg)
		s3Data := []types.Bucket{}
		paginator := s3.NewListBucketsPaginator(s3_client, &s3.ListBucketsInput{})
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Buckets) == 0 {
				continue outer
			}
			for _, bucket := range page.Buckets {
				s3Data = append(s3Data, bucket)
			}
		}
		buckets = append(buckets, getBucketData(s3Data, *s3_client, *cw_client, name, region)...)
	}
	return buckets
}
