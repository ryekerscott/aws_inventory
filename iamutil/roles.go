package iamutil

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	ltypes "repo.com/path/inventory/types"
)

func getRoleData(data []types.Role, iam_client iam.Client, account string) [][]string {
	var roleArr [][]string
	for _, i := range data {
		var costCenter, project string
		polTag, err := iam_client.ListRoleTags(
			context.TODO(),
			&iam.ListRoleTagsInput{RoleName: i.RoleName},
		)
		if err != nil {
			costCenter, project = "N/A", "N/A"
		} else {
			tags := processTags(polTag.Tags)
			costCenter = tags["CostCenter"]
			project = tags["Project"]
		}
		// lastUsed := "N/A"
		// if i.RoleLastUsed != nil {
		// 	lastUsed = fmt.Sprintf("%s", i.RoleLastUsed.LastUsedDate.Format("2006-01-02"))
		// }

		role := []string{
			account,
			aws.ToString(i.RoleName),
			// lastUsed,
			costCenter,
			project,
		}
		roleArr = append(roleArr, role)
	}
	return roleArr
}

func Inventory_Roles(credMap map[string]ltypes.Env) [][]string {
	roles := [][]string{
		{
			"Account",
			"Name",
			"Last Used",
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
		roleData := []types.Role{}
		paginator := iam.NewListRolesPaginator(
			iam_client,
			&iam.ListRolesInput{},
		)
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Roles) == 0 {
				continue outer
			}
			for _, res := range page.Roles {
				roleData = append(roleData, res)
			}
		}
		roles = append(roles, getRoleData(roleData, *iam_client, name)...)
	}
	return roles

}
