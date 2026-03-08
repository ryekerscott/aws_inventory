package iamutil

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getPolicyData(data []types.Policy, iam_client iam.Client, account string) [][]string {
	var policyArr [][]string
	for _, i := range data {
		var costCenter, project string
		polTag, err := iam_client.ListPolicyTags(
			context.TODO(),
			&iam.ListPolicyTagsInput{PolicyArn: i.Arn},
		)
		if err != nil {
			costCenter, project = "N/A", "N/A"
		} else {
			tags := processTags(polTag.Tags)
			costCenter = tags["CostCenter"]
			project = tags["Project"]
		}

		policy := []string{
			account,
			aws.ToString(i.PolicyName),
			costCenter,
			project,
		}
		policyArr = append(policyArr, policy)
	}
	return policyArr
}

func Inventory_Policies(credMap map[string]ltypes.Env) [][]string {
	policies := [][]string{
		{
			"Account",
			"Name",
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
		if v.Skip_iam {
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
		iam_client := iam.NewFromConfig(cfg)
		policyData := []types.Policy{}
		paginator := iam.NewListPoliciesPaginator(
			iam_client,
			&iam.ListPoliciesInput{Scope: types.PolicyScopeTypeLocal},
		)
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Policies) == 0 {
				continue outer
			}
			for _, res := range page.Policies {
				policyData = append(policyData, res)
			}
		}
		policies = append(policies, getPolicyData(policyData, *iam_client, name)...)
	}
	return policies

}
