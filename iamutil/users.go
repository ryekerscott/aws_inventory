package iamutil

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	ltypes "repo.com/path/inventory/types"
)

func getUserData(data []types.User, iam_client iam.Client, account string) [][]string {
	var userArr [][]string
	for _, i := range data {
		groupArr := []string{}
		groups, err := iam_client.ListGroupsForUser(
			context.TODO(),
			&iam.ListGroupsForUserInput{UserName: i.UserName},
		)
		if err != nil {
			groupArr = append(groupArr, "N/A")
		} else {
			for _, x := range groups.Groups {
				groupArr = append(groupArr, *x.GroupName)
			}
		}
		pwd := "N/A"
		if i.PasswordLastUsed != nil {
			pwd = fmt.Sprintf("%s", i.PasswordLastUsed.Format("2006-01-02"))
		}
		groupString := strings.Join(groupArr, ",")
		role := []string{
			account,
			aws.ToString(i.UserName),
			fmt.Sprintf("%s", i.CreateDate.Format("2006-01-02")),
			pwd,
			groupString,
		}
		userArr = append(userArr, role)
	}
	return userArr
}

func Inventory_Users(credMap map[string]ltypes.Env) [][]string {
	users := [][]string{
		{
			"Account",
			"Username",
			"Creation",
			"Password Last Used",
			"Groups",
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
		userData := []types.User{}
		paginator := iam.NewListUsersPaginator(
			iam_client,
			&iam.ListUsersInput{},
		)
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Users) == 0 {
				continue outer
			}
			for _, res := range page.Users {
				userData = append(userData, res)
			}
		}
		users = append(users, getUserData(userData, *iam_client, name)...)
	}
	return users

}
