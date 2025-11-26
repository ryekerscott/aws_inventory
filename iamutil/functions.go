package iamutil

import (
	"slices"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func processTags(rolTags []types.Tag) map[string]string {
	tags := map[string]string{"CostCenter": "N/A", "Project": "N/A"}
	tagsToFind := []string{"CostCenter", "Project"}
	for _, tag := range rolTags {
		if slices.Contains(tagsToFind, string(*tag.Key)) {
			tags[string(*tag.Key)] = string(*tag.Value)
		}
	}
	return tags
}
