package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
	"github.com/ryekerscott/aws_inventory/acmutil"
	"github.com/ryekerscott/aws_inventory/apiutil"
	"github.com/ryekerscott/aws_inventory/cloudwatchutil"
	dynamodbutil "github.com/ryekerscott/aws_inventory/dyanmodbutil"
	"github.com/ryekerscott/aws_inventory/ec2util"
	"github.com/ryekerscott/aws_inventory/efsutil"
	"github.com/ryekerscott/aws_inventory/eksutil"
	"github.com/ryekerscott/aws_inventory/emrutil"
	"github.com/ryekerscott/aws_inventory/iamutil"
	"github.com/ryekerscott/aws_inventory/lambdautil"
	"github.com/ryekerscott/aws_inventory/osutil"
	"github.com/ryekerscott/aws_inventory/rdsutil"
	"github.com/ryekerscott/aws_inventory/route53util"
	"github.com/ryekerscott/aws_inventory/s3util"
	"github.com/ryekerscott/aws_inventory/snsutil"
	"github.com/ryekerscott/aws_inventory/sqsutil"
	"github.com/ryekerscott/aws_inventory/types"
	"github.com/ryekerscott/aws_inventory/wsutil"
)

func writeInventoryFile(file *excelize.File, filename string, sheetName string, message string, data [][]string, newSheet bool, renameSheet1 bool) {
	if newSheet {
		file.NewSheet(sheetName)
	}
	if renameSheet1 {
		file.SetSheetName("Sheet1", sheetName)
	}
	if message != "" {
		fmt.Println(message+": ", len(data)-1)
	}
	for i, v := range data {
		file.SetSheetRow(sheetName, fmt.Sprintf("A%d", i+1), &v)
	}
	rows, _ := file.GetRows(sheetName)
	rCount := len(rows)
	cCount := len(rows[len(rows)-1])
	endCell, _ := excelize.CoordinatesToCellName(cCount, rCount)

	file.AddTable(sheetName, &excelize.Table{
		Range:     fmt.Sprintf("A1:%s", endCell),
		Name:      sheetName,
		StyleName: "TableStyleMedium2",
	})
	file.SaveAs(filename)
}

func runInventory(inventory string, f func(map[string]types.Env) [][]string, creds map[string]types.Env, out chan<- [][]string) {
	fmt.Printf("Inventory %s...\n", inventory)
	out <- f(creds)
	fmt.Printf("%s Complete!\n", inventory)
}

func credentialHandler(config *types.Config) *types.Config {
	for k := range config.Environments {
		access := os.Getenv(config.Environments[k].Credentials["access"])
		secret := os.Getenv(config.Environments[k].Credentials["secret"])
		if access != "" && secret != "" {
			config.Environments[k].Credentials["access"] = access
			config.Environments[k].Credentials["secret"] = secret
		} else {
			fmt.Printf("Credentials not found for environment: %s. Skipping...\n", config.Environments[k].Name)
			if k == config.Output.BucketEnv && config.Output.Upload {
				fmt.Println("Upload Env Credentials Missing! Upload Disabled, Local File Output Only!!!")
				config.Output.Upload = false
			}
			delete(config.Environments, k)
		}
	}
	return config
}

func loadConfig() types.Config {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	var config types.Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}
	config = *credentialHandler(&config)
	return config
}

func main() {
	userConfig := loadConfig()
	creds := userConfig.Environments
	inv := userConfig.Contents
	file := excelize.NewFile()
	currentTime := time.Now()
	filename := fmt.Sprintf("%s-%d-%02d-%d.xlsx", userConfig.Output.Prefix, currentTime.Year(), int(currentTime.Month()), int(currentTime.Day()))
	var inventories []*types.ServiceInventory
	// var inventoryContents [][][]string
	if inv.EC2 {
		ec2 := &types.ServiceInventory{
			Name:    "EC2",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "EC2 Instances",
		}
		ami := &types.ServiceInventory{
			Name:    "AMI",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "AMIs",
		}
		go runInventory(ec2.Message, ec2util.Inventory_EC2, creds, ec2.Channel)
		go runInventory(ami.Message, ec2util.Inventory_AMI, creds, ami.Channel)
		inventories = append(inventories, ec2, ami)
	}
	if inv.EMR {
		emr := &types.ServiceInventory{
			Name:    "EMR",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "EMR Clusters",
		}
		go runInventory(emr.Message, emrutil.Inventory, creds, emr.Channel)
		inventories = append(inventories, emr)
	}
	if inv.RDS {
		rds := &types.ServiceInventory{
			Name:    "RDS",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "RDS Instances",
		}
		go runInventory(rds.Message, rdsutil.Inventory, creds, rds.Channel)
		inventories = append(inventories, rds)
	}
	if inv.WORKSPACES {
		workspaces := &types.ServiceInventory{
			Name:    "Workspaces",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "Workspaces",
		}
		go runInventory(workspaces.Message, wsutil.Inventory, creds, workspaces.Channel)
		inventories = append(inventories, workspaces)
	}
	if inv.ROUTE53 {
		r53 := &types.ServiceInventory{
			Name:    "Route53",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "Route53 Records",
		}
		go runInventory(r53.Message, route53util.Inventory, creds, r53.Channel)
		inventories = append(inventories, r53)
	}
	if inv.ACM {
		acm := &types.ServiceInventory{
			Name:    "ACM",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "ACM Certificates",
		}
		go runInventory(acm.Message, acmutil.Inventory, creds, acm.Channel)
		inventories = append(inventories, acm)
	}
	if inv.DYNAMODB {
		tables := &types.ServiceInventory{
			Name:    "DynamoDB",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "DynamoDB Tables",
		}
		go runInventory(tables.Message, dynamodbutil.Inventory, creds, tables.Channel)
		inventories = append(inventories, tables)
	}
	if inv.S3 {
		s3 := &types.ServiceInventory{
			Name:    "S3",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "S3 Buckets",
		}
		go runInventory(s3.Message, s3util.Inventory, creds, s3.Channel)
		inventories = append(inventories, s3)
	}
	if inv.ELB {
		elb := &types.ServiceInventory{
			Name:    "ELB",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "Elastic Loadbalancers",
		}
		go runInventory(elb.Message, ec2util.Inventory_ELB, creds, elb.Channel)
		inventories = append(inventories, elb)
	}
	if inv.OPENSEARCH {
		opensearch := &types.ServiceInventory{
			Name:    "Opensearch",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "Opensearch Clusters",
		}
		go runInventory(opensearch.Message, osutil.Inventory, creds, opensearch.Channel)
		inventories = append(inventories, opensearch)
	}
	if inv.APIGW {
		api := &types.ServiceInventory{
			Name:    "APIGW",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "API Gateways",
		}
		go runInventory(api.Message, apiutil.Inventory, creds, api.Channel)
		inventories = append(inventories, api)
	}
	if inv.EBS {
		ebs := &types.ServiceInventory{
			Name:    "EBS",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "EBS Volumes",
		}
		go runInventory(ebs.Message, ec2util.Inventory_EBS, creds, ebs.Channel)
		inventories = append(inventories, ebs)
	}
	if inv.LAMBDA {
		lambda := &types.ServiceInventory{
			Name:    "Lambda",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "Lambda Functions",
		}
		go runInventory(lambda.Message, lambdautil.Inventory, creds, lambda.Channel)
		inventories = append(inventories, lambda)
	}
	if inv.IAM {
		roles := &types.ServiceInventory{
			Name:    "Roles",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "IAM Roles",
		}
		policies := &types.ServiceInventory{
			Name:    "Policies",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "IAM Policies",
		}
		users := &types.ServiceInventory{
			Name:    "Users",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "IAM Users",
		}
		groups := &types.ServiceInventory{
			Name:    "Groups",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "IAM Groups",
		}
		go runInventory(policies.Message, iamutil.Inventory_Policies, creds, policies.Channel)
		go runInventory(roles.Message, iamutil.Inventory_Roles, creds, roles.Channel)
		go runInventory(users.Message, iamutil.Inventory_Users, creds, users.Channel)
		go runInventory(groups.Message, iamutil.Inventory_Groups, creds, groups.Channel)
		inventories = append(inventories, roles, policies, users, groups)
	}
	if inv.EFS {
		efs := &types.ServiceInventory{
			Name:    "EFS",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "EFS",
		}
		go runInventory(efs.Message, efsutil.Inventory, creds, efs.Channel)
		inventories = append(inventories, efs)
	}
	if inv.EKS {
		eks := &types.ServiceInventory{
			Name:    "EKS",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "EKS Clusters",
		}
		go runInventory(eks.Message, eksutil.Inventory, creds, eks.Channel)
		inventories = append(inventories, eks)
	}
	if inv.VPC {
		vpc := &types.ServiceInventory{
			Name:    "VPC",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "VPCs",
		}
		subnets := &types.ServiceInventory{
			Name:    "Subnets",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "Subnets",
		}
		rtbs := &types.ServiceInventory{
			Name:    "RTBs",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "Route Tables",
		}
		nacls := &types.ServiceInventory{
			Name:    "NACLs",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "Network ACLs",
		}
		sgs := &types.ServiceInventory{
			Name:    "SGs",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "Security Groups",
		}
		go runInventory(vpc.Message, ec2util.Inventory_VPC, creds, vpc.Channel)
		go runInventory(subnets.Message, ec2util.Inventory_Subnets, creds, subnets.Channel)
		go runInventory(rtbs.Message, ec2util.Inventory_RTB, creds, rtbs.Channel)
		go runInventory(nacls.Message, ec2util.Inventory_Nacl, creds, nacls.Channel)
		go runInventory(sgs.Message, ec2util.Inventory_Sg, creds, sgs.Channel)
		inventories = append(inventories, vpc, subnets, rtbs, nacls, sgs)
	}
	if inv.SQS {
		queues := &types.ServiceInventory{
			Name:    "SQS",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "SQS Queues",
		}
		go runInventory(queues.Message, sqsutil.Inventory, creds, queues.Channel)
		inventories = append(inventories, queues)
	}
	if inv.SNS {
		topics := &types.ServiceInventory{
			Name:    "SNS",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "SNS Topics",
		}
		go runInventory(topics.Message, snsutil.Inventory, creds, topics.Channel)
		inventories = append(inventories, topics)
	}
	if inv.CLOUDWATCH {
		logGroups := &types.ServiceInventory{
			Name:    "Cloudwatch",
			Channel: make(chan [][]string, 1),
			Values:  [][]string{},
			Message: "Cloudwatch Log Groups",
		}
		go runInventory(logGroups.Message, cloudwatchutil.Inventory, creds, logGroups.Channel)
		inventories = append(inventories, logGroups)
	}

	for _, v := range inventories {
		v.Values = <-v.Channel
	}
	summary := [][]string{
		{
			"Resource",
			"Count",
		},
	}
	for i := range inventories {
		if len(inventories[i].Values) > 1 {
			summary = append(summary, []string{inventories[i].Message, fmt.Sprintf("%d", len(inventories[i].Values)-1)})
		}
	}
	writeInventoryFile(file, filename, "Summary", "", summary, false, true)
	fmt.Println("Inventory Contents:")
	for i := range inventories {
		if len(inventories[i].Values) > 1 {
			writeInventoryFile(file, filename, inventories[i].Name, inventories[i].Message, inventories[i].Values, true, false)
		}
	}
	if userConfig.Output.Upload {
		ctx := context.TODO()
		output_env := userConfig.Environments[userConfig.Output.BucketEnv]
		access := output_env.Credentials["access"]
		secret := output_env.Credentials["secret"]
		provider := credentials.NewStaticCredentialsProvider(access, secret, "")
		cfg, err := config.LoadDefaultConfig(
			ctx,
			config.WithCredentialsProvider(provider),
			config.WithRegion(output_env.Credentials["region"]),
		)
		if err != nil {
			panic("Failed to upload to s3: Authentication error: " + err.Error())
		}
		s3_client := s3.NewFromConfig(cfg)
		bucketName := userConfig.Output.BucketName
		s3File, _ := os.Open(filename)
		_, err = s3_client.PutObject(ctx, &s3.PutObjectInput{Bucket: &bucketName, Key: &filename, Body: s3File})
		if err != nil {
			panic("Failed to Upload to s3: PutObject Error: " + err.Error())
		} else {
			fmt.Printf("Uploaded to S3: %s bucket in %s account.\n", bucketName, userConfig.Environments[userConfig.Output.BucketEnv].Name)
		}
	} else {
		fmt.Println("Skipping Upload!")
	}
}
