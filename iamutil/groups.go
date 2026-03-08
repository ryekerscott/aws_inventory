package iamutil

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getGroupData(data []types.Group, iam_client iam.Client, account string) [][]string {
	var groupArr [][]string
	for _, i := range data {
		groupDetail, _ := iam_client.GetGroup(context.TODO(), &iam.GetGroupInput{GroupName: *&i.GroupName})
		numUsersInGroup := len(groupDetail.Users)
		group := []string{
			account,
			aws.ToString(i.GroupName),
			fmt.Sprintf("%d", numUsersInGroup),
		}
		groupArr = append(groupArr, group)
	}
	return groupArr
}

func Inventory_Groups(credMap map[string]ltypes.Env) [][]string {
	groups := [][]string{
		{
			"Account",
			"Name",
			"Group Users",
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
		groupData := []types.Group{}
		paginator := iam.NewListGroupsPaginator(
			iam_client,
			&iam.ListGroupsInput{},
		)
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Groups) == 0 {
				continue outer
			}
			for _, res := range page.Groups {
				groupData = append(groupData, res)
			}
		}
		groups = append(groups, getGroupData(groupData, *iam_client, name)...)
	}
	return groups

}
