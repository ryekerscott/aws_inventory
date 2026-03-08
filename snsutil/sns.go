package snsutil

import (
	"context"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getTopicData(data []types.Topic, sns_client sns.Client, account string, region string) [][]string {
	var snsArr [][]string
	for _, i := range data {
		topic_attr, _ := sns_client.GetTopicAttributes(
			context.TODO(),
			&sns.GetTopicAttributesInput{
				TopicArn: i.TopicArn,
			},
		)
		nameParts := strings.Split(*i.TopicArn, ":")
		name := nameParts[len(nameParts)-1]
		tagOut, _ := sns_client.ListTagsForResource(context.TODO(), &sns.ListTagsForResourceInput{ResourceArn: i.TopicArn})
		topicTags := tagOut.Tags
		tagsToFind := []string{"CostCenter", "Project"}
		tags := map[string]string{"CostCenter": "N/A", "Project": "N/A"}
		for _, val := range topicTags {
			if slices.Contains(tagsToFind, *val.Key) {
				tags[*val.Key] = *val.Value
			}
		}
		topic := []string{
			account,
			region,
			name,
			topic_attr.Attributes["SubscriptionsConfirmed"],
			topic_attr.Attributes["SubscriptionsPending"],
			aws.ToString(i.TopicArn),
			tags["CostCenter"],
			tags["Project"],
		}

		snsArr = append(snsArr, topic)
	}
	return snsArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	topics := [][]string{
		{
			"Account",
			"Region",
			"Name",
			"Subscriptions Confirmed",
			"Subscriptions Pending",
			"ARN",
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
		sns_client := sns.NewFromConfig(cfg)
		topicData := []types.Topic{}
		paginator := sns.NewListTopicsPaginator(sns_client, &sns.ListTopicsInput{})
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Topics) == 0 {
				continue outer
			}
			for _, res := range page.Topics {
				// queueAttr, _ := sqs_client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{QueueUrl: &res})
				topicData = append(topicData, res)
			}
		}
		topics = append(topics, getTopicData(topicData, *sns_client, name, region)...)
	}
	return topics
}
