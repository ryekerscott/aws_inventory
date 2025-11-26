package sqsutil

import (
	"context"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	ltypes "repo.com/path/inventory/types"
)

func getQueueData(data []string, sqs_client sqs.Client, account string, region string) [][]string {
	var sqsArr [][]string
	for _, i := range data {
		tagOut, _ := sqs_client.ListQueueTags(context.TODO(), &sqs.ListQueueTagsInput{QueueUrl: &i})
		queueTags := tagOut.Tags
		tagsToFind := []string{"CostCenter", "Project"}
		tags := map[string]string{"CostCenter": "N/A", "Project": "N/A"}
		urlSplit := strings.Split(i, "/")
		name := urlSplit[len(urlSplit)-1]
		for tag, val := range queueTags {
			if slices.Contains(tagsToFind, tag) {
				tags[tag] = val
			}
		}
		attrs, err := sqs_client.GetQueueAttributes(
			context.TODO(),
			&sqs.GetQueueAttributesInput{
				QueueUrl:       &i,
				AttributeNames: []types.QueueAttributeName{"All"},
			},
		)
		if err != nil {
			panic(err.Error())
		}
		attributes := attrs.Attributes
		queueType := "Standard"
		if attributes["FifoQueue"] == "true" {
			queueType = "FIFO"
		}
		queue := []string{
			account,
			region,
			name,
			queueType,
			attributes["ApproximateNumberOfMessages"],
			attributes["QueueArn"],
			tags["CostCenter"],
			tags["Project"],
		}

		sqsArr = append(sqsArr, queue)
	}
	return sqsArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	queues := [][]string{
		{
			"Account",
			"Region",
			"Queue Name",
			"Queue Type",
			"Approx. Num of Messages",
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
		sqs_client := sqs.NewFromConfig(cfg)
		queueData := []string{}
		paginator := sqs.NewListQueuesPaginator(sqs_client, &sqs.ListQueuesInput{})
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.QueueUrls) == 0 {
				continue outer
			}
			for _, res := range page.QueueUrls {
				// queueAttr, _ := sqs_client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{QueueUrl: &res})
				queueData = append(queueData, res)
			}
		}
		queues = append(queues, getQueueData(queueData, *sqs_client, name, region)...)
	}
	return queues
}
