package ec2util

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getImageData(data []types.Image, ec2_client ec2.Client, account string, region string) [][]string {
	var amiArr [][]string
	for _, i := range data {
		name := aws.ToString(i.Name)
		if strings.HasPrefix(name, "AwsBackup") {
			continue
		}
		ami := aws.ToString(i.ImageId)
		image := []string{
			account,
			region,
			aws.ToString(i.Name),
			ami,
			string(i.Architecture),
			string(*i.PlatformDetails),
			aws.ToString(i.CreationDate),
		}
		amiArr = append(amiArr, image)
	}
	return amiArr
}

func Inventory_AMI(credMap map[string]ltypes.Env) [][]string {
	images := [][]string{
		{
			"Account",
			"Region",
			"AMI Name",
			"AMI ID",
			"Architecture",
			"Operating System",
			"Creation Date",
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
		ec2_client := ec2.NewFromConfig(cfg)
		imageData := []types.Image{}
		paginator := ec2.NewDescribeImagesPaginator(ec2_client, &ec2.DescribeImagesInput{ExecutableUsers: []string{"self"}})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				panic(err.Error())
			}
			if len(page.Images) == 0 {
				continue outer
			}
			for _, res := range page.Images {
				imageData = append(imageData, res)
			}
		}
		images = append(images, getImageData(imageData, *ec2_client, name, region)...)
	}
	return images

}
