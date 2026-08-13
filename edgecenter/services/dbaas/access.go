package dbaas

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const dbaasClusterAccessPollInterval = 5 * time.Second

func expandDBaaSClusterAccess(raw interface{}) *edgecloudV2.DBaaSClusterAccess {
	accessList, ok := raw.([]interface{})
	if !ok || len(accessList) == 0 || accessList[0] == nil {
		return nil
	}

	accessMap, ok := accessList[0].(map[string]interface{})
	if !ok {
		return nil
	}

	return &edgecloudV2.DBaaSClusterAccess{
		AllowedCIDRs: expandStringSet(accessMap[edgecenter.DBaaSClusterAllowedCIDRsField]),
		IsPublic:     accessMap[edgecenter.DBaaSClusterIsPublicField].(bool),
	}
}

func expandDBaaSClusterAccessControlUpdateRequest(raw interface{}) edgecloudV2.DBaaSClusterAccessControlUpdateRequest {
	access := expandDBaaSClusterAccess(raw)
	if access == nil {
		emptyCIDRs := []string{}
		isPublic := false

		return edgecloudV2.DBaaSClusterAccessControlUpdateRequest{
			AllowedCIDRs: &emptyCIDRs,
			IsPublic:     &isPublic,
		}
	}

	allowedCIDRs := access.AllowedCIDRs
	isPublic := access.IsPublic

	return edgecloudV2.DBaaSClusterAccessControlUpdateRequest{
		AllowedCIDRs: &allowedCIDRs,
		IsPublic:     &isPublic,
	}
}

func flattenDBaaSClusterAccess(access *edgecloudV2.DBaaSClusterAccess) []interface{} {
	if access == nil {
		return nil
	}

	return []interface{}{
		map[string]interface{}{
			edgecenter.DBaaSClusterAllowedCIDRsField: schema.NewSet(schema.HashString, stringSliceToInterfaceSlice(access.AllowedCIDRs)),
			edgecenter.DBaaSClusterIsPublicField:     access.IsPublic,
		},
	}
}

func flattenDBaaSClusterConnection(connection *edgecloudV2.DBaaSConnection) []interface{} {
	if connection == nil {
		return nil
	}

	return []interface{}{
		map[string]interface{}{
			edgecenter.DBaaSClusterHostField: connection.Host,
			edgecenter.DBaaSClusterPortField: connection.Port,
		},
	}
}

func expandStringSet(raw interface{}) []string {
	set, ok := raw.(*schema.Set)
	if !ok || set == nil {
		return nil
	}

	values := set.List()
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.(string))
	}

	return result
}

func waitDBaaSClusterAccessState(
	ctx context.Context,
	client *edgecloudV2.Client,
	clusterID string,
	expectedAccess *edgecloudV2.DBaaSClusterAccess,
	timeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(dbaasClusterAccessPollInterval)
	defer ticker.Stop()

	for {
		cluster, _, err := client.DBaaS.ClusterGet(ctx, clusterID)
		if err != nil {
			return fmt.Errorf("get DBaaS cluster %s: %w", clusterID, err)
		}

		if isDBaaSClusterAccessReady(cluster, expectedAccess) {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("DBaaS cluster access state did not converge within timeout")
		case <-ticker.C:
		}
	}
}

func isDBaaSClusterAccessReady(cluster *edgecloudV2.DBaaSCluster, expectedAccess *edgecloudV2.DBaaSClusterAccess) bool {
	if cluster == nil || cluster.Access == nil {
		return false
	}

	if cluster.Access.IsPublic != expectedAccess.IsPublic {
		return false
	}

	if !stringSetsEqual(cluster.Access.AllowedCIDRs, expectedAccess.AllowedCIDRs) {
		return false
	}

	return true
}

func stringSetsEqual(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}

	actualSet := make(map[string]struct{}, len(actual))
	for _, value := range actual {
		actualSet[value] = struct{}{}
	}

	for _, value := range expected {
		if _, ok := actualSet[value]; !ok {
			return false
		}
	}

	return true
}

func stringSliceToInterfaceSlice(values []string) []interface{} {
	result := make([]interface{}, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}

	return result
}
