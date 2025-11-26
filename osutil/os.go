package osutil

import (
	"context"
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	es "github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice/types"
	ltypes "repo.com/path/inventory/types"
)

func getCustomEndpointArn(certArn string, acm_client acm.Client) string {
	cert, _ := acm_client.DescribeCertificate(
		context.TODO(),
		&acm.DescribeCertificateInput{
			CertificateArn: &certArn,
		},
	)
	return aws.ToString(cert.Certificate.CertificateArn)

}

func getDomainTags(osArn string, os_client es.Client) map[string]string {
	tags := map[string]string{"CostCenter": "N/A", "Project": "N/A"}
	tagsToFind := []string{"CostCenter", "Project"}
	domainTags, _ := os_client.ListTags(
		context.TODO(),
		&es.ListTagsInput{
			ARN: &osArn,
		},
	)
	for _, tag := range domainTags.TagList {
		if slices.Contains(tagsToFind, string(*tag.Key)) {
			tags[string(*tag.Key)] = string(*tag.Value)
		}
	}
	return tags

}

func getDomainData(data []types.ElasticsearchDomainStatus, os_client es.Client, acm_client acm.Client, account string, region string) [][]string {
	var domainArr [][]string
	for _, i := range data {
		customEndpoint := ""
		customEndpointArn := ""
		if *i.DomainEndpointOptions.CustomEndpointEnabled {
			customEndpoint = aws.ToString(i.DomainEndpointOptions.CustomEndpoint)
			customEndpointArn = getCustomEndpointArn(
				aws.ToString(i.DomainEndpointOptions.CustomEndpointCertificateArn),
				acm_client,
			)
		}
		tags := getDomainTags(*i.ARN, os_client)
		domain := []string{
			account,
			aws.ToString(i.DomainName),
			region,
			aws.ToString(i.ElasticsearchVersion),
			customEndpoint,
			customEndpointArn,
			string(i.ElasticsearchClusterConfig.DedicatedMasterType),
			fmt.Sprintf("%d", aws.ToInt32(i.ElasticsearchClusterConfig.DedicatedMasterCount)),
			string(i.ElasticsearchClusterConfig.InstanceType),
			fmt.Sprintf("%d", aws.ToInt32(i.ElasticsearchClusterConfig.InstanceCount)),
			tags["CostCenter"],
			tags["Project"],
		}
		domainArr = append(domainArr, domain)
	}
	return domainArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	clusters := [][]string{
		{
			"Account",
			"Domain",
			"Region",
			"Version",
			"Custom Endpoint",
			"Cert ARN",
			"Primary Node Type",
			"Num Primary Nodes",
			"Data Node Type",
			"Num Data Nodes",
			"CostCenter",
			"Project",
		},
	}
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
		os_client := es.NewFromConfig(cfg)
		acm_client := acm.NewFromConfig(cfg)
		dlo, _ := os_client.ListDomainNames(
			ctx,
			&es.ListDomainNamesInput{},
		)
		domains := []types.ElasticsearchDomainStatus{}
		for _, x := range dlo.DomainNames {
			domain, _ := os_client.DescribeElasticsearchDomain(
				ctx,
				&es.DescribeElasticsearchDomainInput{
					DomainName: x.DomainName,
				},
			)
			domains = append(domains, *domain.DomainStatus)
		}
		// domains, err := os_client.DescribeElasticsearchDomains(
		// 	ctx,
		// 	&es.DescribeElasticsearchDomainsInput{DomainNames: domainsList},
		// )
		if len(domains) != 0 {
			clusters = append(clusters, getDomainData(domains, *os_client, *acm_client, name, region)...)
		}
	}
	return clusters
}
