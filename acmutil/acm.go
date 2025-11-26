package acmutil

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	ltypes "repo.com/path/inventory/types"
)

func getCertificateData(data []types.CertificateDetail, account string) [][]string {
	var certArr [][]string
	for _, i := range data {
		NA := i.NotAfter
		notAfterString := ""
		daysUntilExpiration := "N/A"
		if NA != nil {
			notAfterString = fmt.Sprintf("%s %d, %d", NA.Month(), NA.Day(), NA.Year())
			daysUntil := int(time.Now().Sub(*NA).Hours()/24) * -1
			if daysUntil < 0 {
				daysUntilExpiration = "Expired"
			} else {
				daysUntilExpiration = fmt.Sprintf("%d", daysUntil)
			}
		} else {
			notAfterString = "N/A"
		}
		sans := "N/A"
		if len(i.SubjectAlternativeNames) > 0 {
			sans = ""
			for _, san := range i.SubjectAlternativeNames {
				sans = sans + san + ", "
			}
			sans, _ = strings.CutSuffix(sans, ", ")
		}
		usedBy := "False"
		if len(i.InUseBy) > 0 {
			usedBy = "True"
		}

		certificate := []string{
			account,
			aws.ToString(i.DomainName),
			sans,
			aws.ToString(i.CertificateArn),
			usedBy,
			notAfterString,
			daysUntilExpiration,
		}
		certArr = append(certArr, certificate)
	}
	return certArr
}

func Inventory(credMap map[string]ltypes.Env) [][]string {
	certificates := [][]string{
		{
			"Account",
			"Subject",
			"SANs",
			"ARN",
			"In Use",
			"Expiration",
			"Days Until Expiration",
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
		acm_client := acm.NewFromConfig(cfg)
		certificateData := []types.CertificateDetail{}
		paginator := acm.NewListCertificatesPaginator(
			acm_client,
			&acm.ListCertificatesInput{
				Includes: &types.Filters{
					KeyTypes: []types.KeyAlgorithm{
						"RSA_1024",
						"RSA_2048",
						"RSA_4096",
					},
				},
			},
		)
		for paginator.HasMorePages() {
			page, _ := paginator.NextPage(ctx)
			if len(page.CertificateSummaryList) == 0 {
				continue outer
			}
			for _, cert := range page.CertificateSummaryList {
				certData, _ := acm_client.DescribeCertificate(
					ctx,
					&acm.DescribeCertificateInput{
						CertificateArn: cert.CertificateArn,
					},
				)
				certificateData = append(certificateData, *certData.Certificate)
			}
		}
		certificates = append(certificates, getCertificateData(certificateData, name)...)
	}
	return certificates

}
