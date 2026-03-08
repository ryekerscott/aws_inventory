package route53util

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getRecordData(data []types.ResourceRecordSet, hzs []string, r53_client route53.Client, account string) [][]string {
	var recordArr [][]string
	for x, i := range data {
		targetArr := []string{}
		for j := range i.ResourceRecords {
			targetArr = append(targetArr, *i.ResourceRecords[j].Value)
		}
		targets := strings.Join(targetArr, ", ")
		record := []string{
			account,
			aws.ToString(i.Name),
			hzs[x],
			string(i.Type),
			targets,
		}
		recordArr = append(recordArr, record)
	}
	return recordArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	records := [][]string{
		{
			"Account",
			"Record",
			"Hosted Zone",
			"Type",
			"Targets",
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
		if v.Skip_r53 {
			continue outer
		}
		name := v.Name
		ctx := context.TODO()
		provider := credentials.NewStaticCredentialsProvider(access, secret, "")
		cfg, _ := config.LoadDefaultConfig(
			ctx,
			config.WithCredentialsProvider(provider),
			config.WithRegion(region),
		)
		r53_client := route53.NewFromConfig(cfg)
		recordData := []types.ResourceRecordSet{}
		hzData, _ := r53_client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
		hzs := []string{}
		for _, v := range hzData.HostedZones {

			paginator := route53.NewListResourceRecordSetsPaginator(
				r53_client,
				&route53.ListResourceRecordSetsInput{
					HostedZoneId: v.Id,
				},
			)
			for paginator.HasMorePages() {
				page, _ := paginator.NextPage(ctx)
				if len(page.ResourceRecordSets) == 0 {
					continue outer
				}
				for _, res := range page.ResourceRecordSets {
					hzs = append(hzs, aws.ToString(v.Name))
					recordData = append(recordData, res)
				}
			}
			records = append(records, getRecordData(recordData, hzs, *r53_client, name)...)
		}

	}
	return records

}
