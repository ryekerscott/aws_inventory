package ec2util

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	ltypes "github.com/ryekerscott/aws_inventory/types"
)

func getListenerAttributes(lbArn string, elb_client elasticloadbalancingv2.Client, acm_client acm.Client) map[string]map[string]string {
	ctx := context.TODO()
	listenerArr, _ := elb_client.DescribeListeners(
		ctx,
		&elasticloadbalancingv2.DescribeListenersInput{LoadBalancerArn: &lbArn},
	)
	listeners := listenerArr.Listeners
	listenerMap := make(map[string]map[string]string)
	for _, v := range listeners {
		certArns := ""
		certNames := ""
		daysUntilExpiration := []string{}
		for _, c := range v.Certificates {
			if *c.CertificateArn != "" {
				certArns = certArns + aws.ToString(c.CertificateArn) + ", "
				certData, _ := acm_client.DescribeCertificate(
					ctx,
					&acm.DescribeCertificateInput{
						CertificateArn: c.CertificateArn,
					},
				)
				certNames = certNames + aws.ToString(certData.Certificate.DomainName) + ", "
				certDetail, _ := acm_client.DescribeCertificate(
					ctx,
					&acm.DescribeCertificateInput{
						CertificateArn: c.CertificateArn,
					},
				)
				NA := certDetail.Certificate.NotAfter
				daysUntil := int(time.Now().Sub(*NA).Hours()/24) * -1
				if daysUntil < 0 {
					daysUntilExpiration = append(daysUntilExpiration, "Expired")
				} else {
					daysUntilExpiration = append(daysUntilExpiration, string(daysUntil))
				}

			}
		}
		certArns = strings.TrimSuffix(certArns, ", ")
		certNames = strings.TrimSuffix(certNames, ", ")
		listenerMap[*v.ListenerArn] = map[string]string{
			"Port":         fmt.Sprintf("%d", aws.ToInt32(v.Port)),
			"Protocol":     string(v.Protocol),
			"Certificates": certArns,
			"SANS":         certNames,
			"Expirations":  strings.Join(daysUntilExpiration, ", "),
		}
	}
	return listenerMap
}

func getLoadbalancerTags(lbArn string, elb_client elasticloadbalancingv2.Client) map[string]string {
	lbTags, _ := elb_client.DescribeTags(
		context.TODO(),
		&elasticloadbalancingv2.DescribeTagsInput{
			ResourceArns: []string{lbArn},
		},
	)
	tagVals := lbTags.TagDescriptions[0].Tags
	tagsToFind := []string{"CostCenter", "Project"}
	tags := map[string]string{
		"CostCenter": "N/A",
		"Project":    "N/A",
	}
	for _, v := range tagVals {
		if slices.Contains(tagsToFind, aws.ToString(v.Key)) {
			tags[aws.ToString(v.Key)] = aws.ToString(v.Value)
		}
	}

	return tags
}

func getLbData(data []types.LoadBalancer, elb_client elasticloadbalancingv2.Client, ec2_client ec2.Client, acm_client acm.Client, account string) [][]string {
	var lbArr [][]string
	for _, i := range data {
		vpcId := aws.ToString(i.VpcId)
		vpcName := vpcInfo(ec2_client, vpcId)
		lbArn := aws.ToString(i.LoadBalancerArn)
		listeners := getListenerAttributes(lbArn, elb_client, acm_client)
		listenerNames := ""
		ports := ""
		protoMap := make(map[string]bool)
		certs := ""
		SANS := ""
		protocols := ""
		expirations := ""
		tags := getLoadbalancerTags(lbArn, elb_client)
		for k, v := range listeners {
			listenerNames += k + ", "
			ports += v["Port"] + ", "
			protoMap[v["Protocol"]] = true
			if v["Certificates"] != "" {
				certs += v["Certificates"] + ", "
				SANS += v["SANS"] + ", "
				expirations += v["Expirations"] + ", "
			}
		}
		for k, _ := range protoMap {
			protocols += k + ", "
		}
		lb := []string{
			account,
			aws.ToString(i.LoadBalancerName),
			aws.ToString(i.DNSName),
			string(i.Type),
			vpcId,
			vpcName,
			strings.TrimSuffix(listenerNames, ", "),
			strings.TrimSuffix(ports, ", "),
			strings.TrimSuffix(protocols, ", "),
			strings.TrimSuffix(certs, ", "),
			strings.TrimSuffix(SANS, ", "),
			strings.ReplaceAll(expirations, ", ", ""),
			tags["CostCenter"],
			tags["Project"],
		}
		lbArr = append(lbArr, lb)
	}
	return lbArr
}

func Inventory_ELB(credMap map[string]ltypes.Env) [][]string {
	lbs := [][]string{
		{
			"Account",
			"Name",
			"DNS Name",
			"Type",
			"VPC ID",
			"VPC Name",
			"Listeners",
			"Ports",
			"Protocols",
			"Cert ARNS",
			"SANS",
			"Cert Expires in (days)",
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
		elb_client := elasticloadbalancingv2.NewFromConfig(cfg)
		ec2_client := ec2.NewFromConfig(cfg)
		acm_client := acm.NewFromConfig(cfg)
		lbData := []types.LoadBalancer{}
		paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(elb_client, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.LoadBalancers) == 0 {
				continue outer
			}
			for _, res := range page.LoadBalancers {
				lbData = append(lbData, res)
			}
		}
		lbs = append(lbs, getLbData(lbData, *elb_client, *ec2_client, *acm_client, name)...)
	}
	return lbs

}
