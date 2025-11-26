package wsutil

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/workspaces"
	"github.com/aws/aws-sdk-go-v2/service/workspaces/types"
	ltypes "repo.com/path/inventory/types"
)

func getInstanceData(data []types.Workspace, account string) [][]string {
	var instanceArr [][]string
	for _, i := range data {

		// tags := processTags(i)
		// ami := aws.ToString(i.ImageId)
		// amiInfo := amiInfo(ec2_client, ami)
		// vpcId, subnetId := aws.ToString(i.VpcId), aws.ToString(i.SubnetId)
		// vpcName, subnetName := vpcInfo(ec2_client, vpcId, subnetId)
		computerName := "N/A"
		ipAddress := "N/A"
		if i.ComputerName != nil {
			computerName = *i.ComputerName
		}
		if i.IpAddress != nil {
			ipAddress = *i.IpAddress
		}
		instance := []string{
			account,
			computerName,
			*i.WorkspaceId,
			*i.UserName,
			ipAddress,
			string(i.State),
			string(i.WorkspaceProperties.ComputeTypeName),
			fmt.Sprintf("%dGB", *i.WorkspaceProperties.RootVolumeSizeGib),
			string(i.WorkspaceProperties.OperatingSystemName),
		}
		instanceArr = append(instanceArr, instance)
	}
	return instanceArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	workspaceInstances := [][]string{
		{
			"Account",
			"Computer Name",
			"Workspace ID",
			"Username",
			"IP Address",
			"State",
			"Compute Type",
			"Storage Capacity",
			"Operating System",
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
		ws_client := workspaces.NewFromConfig(cfg)
		workspacesData := []types.Workspace{}
		paginator := workspaces.NewDescribeWorkspacesPaginator(
			ws_client,
			&workspaces.DescribeWorkspacesInput{})
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Workspaces) == 0 {
				continue outer
			}
			for _, ws := range page.Workspaces {
				workspacesData = append(workspacesData, ws)
			}
		}
		workspaceInstances = append(workspaceInstances, getInstanceData(workspacesData, name)...)
	}
	return workspaceInstances
}
