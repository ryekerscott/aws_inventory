package lambdautil

import (
	"context"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	ltypes "repo.com/path/inventory/types"
)

func getFunctionVpcName(vpcId string, ec2_client ec2.Client) string {
	vpcData, err := ec2_client.DescribeVpcs(
		context.TODO(),
		&ec2.DescribeVpcsInput{
			VpcIds: []string{vpcId},
		},
	)
	if err != nil {
		panic(err.Error())
	}
	vpcTags := vpcData.Vpcs[0].Tags
	vpcName := "N/A"
	for _, tag := range vpcTags {
		if *tag.Key == "Name" {
			vpcName = string(*tag.Value)
		}
	}
	return vpcName
}

func processTags(function *lambda.ListTagsOutput, err error) map[string]string {
	tags := map[string]string{"CostCenter": "N/A", "Project": "N/A"}
	if err != nil {
		return tags
	}
	tagsToFind := []string{"CostCenter", "Project"}
	functionTags := function.Tags
	for key, val := range functionTags {
		if slices.Contains(tagsToFind, string(key)) {
			tags[string(key)] = string(val)
		}
	}
	return tags
}

func getFunctionData(data []types.FunctionConfiguration, lambda_client lambda.Client, ec2_client ec2.Client, account string, region string) [][]string {
	var funcArr [][]string
	for _, i := range data {
		vpcId := "N/A"
		vpcName := "N/A"
		if i.VpcConfig != nil && *i.VpcConfig.VpcId != "" {
			vpcId = aws.ToString(i.VpcConfig.VpcId)
			vpcName = getFunctionVpcName(vpcId, ec2_client)
		}
		tags := processTags(
			lambda_client.ListTags(
				context.TODO(),
				&lambda.ListTagsInput{
					Resource: i.FunctionArn,
				},
			),
		)
		function := []string{
			account,
			region,
			aws.ToString(i.FunctionName),
			string(i.Runtime),
			vpcId,
			vpcName,
			tags["CostCenter"],
			tags["Project"],
		}
		funcArr = append(funcArr, function)

	}
	return funcArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	functions := [][]string{
		{
			"Account",
			"Region",
			"Function Name",
			"Runtime",
			"VPC ID",
			"VPC Name",
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
		lambda_client := lambda.NewFromConfig(cfg)
		ec2_client := ec2.NewFromConfig(cfg)
		functionData := []types.FunctionConfiguration{}
		paginator := lambda.NewListFunctionsPaginator(
			lambda_client,
			&lambda.ListFunctionsInput{},
		)
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.Functions) == 0 {
				continue outer
			}
			for _, res := range page.Functions {
				functionData = append(functionData, res)
			}
		}
		functions = append(functions, getFunctionData(functionData, *lambda_client, *ec2_client, name, region)...)
	}
	return functions

}
