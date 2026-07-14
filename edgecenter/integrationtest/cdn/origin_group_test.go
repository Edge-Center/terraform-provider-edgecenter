//go:build integration

package cdn_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercdn-go/origingroups"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	cdnmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cdn/mock"
)

const (
	testOriginGroupID   = 501
	testOriginGroupName = "tf-origin-group"
	testOriginSource    = "example.com"
)

func originGroupConfig(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":                 name,
		"use_next":             true,
		"consistent_balancing": false,
		"origin": []interface{}{
			map[string]interface{}{
				"source":  testOriginSource,
				"enabled": true,
				"backup":  false,
			},
		},
	}
}

func sampleOriginGroup(name string) *origingroups.OriginGroup {
	return &origingroups.OriginGroup{
		ID:      testOriginGroupID,
		Name:    name,
		UseNext: true,
		Origins: []origingroups.Origin{
			{ID: 11, Source: testOriginSource, Enabled: true, Backup: false},
		},
		ConsistentBalancing: false,
	}
}

func originGroupCreateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.OriginGroups.On("Create", mock.Anything,
		mock.MatchedBy(func(req *origingroups.GroupRequest) bool {
			return req.Name == testOriginGroupName &&
				req.UseNext &&
				!req.ConsistentBalancing &&
				req.Authorization == nil &&
				len(req.Origins) == 1 &&
				req.Origins[0].Source == testOriginSource &&
				req.Origins[0].Enabled &&
				!req.Origins[0].Backup
		}),
	).Return(sampleOriginGroup(testOriginGroupName), nil)

	mc.OriginGroups.On("Get", mock.Anything, int64(testOriginGroupID)).
		Return(sampleOriginGroup(testOriginGroupName), nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: originGroupConfig(testOriginGroupName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testOriginGroupID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":                 testOriginGroupName,
				"use_next":             "true",
				"consistent_balancing": "false",
				"origin.#":             "1",
				"authorization.#":      "0",
			})
		},
	}
}

func originGroupCreateWithAuthCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	auth := &origingroups.Authorization{
		AuthType:        "aws_signature_v4",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		AddressingStyle: "path",
		AwsRegion:       "us-east-1",
		SecretKey:       "wJalrXUtnFEMI-K7MDENG-bPxRfiCYEXAMPLEKEY",
		BucketName:      "tf-bucket",
	}

	group := sampleOriginGroup(testOriginGroupName)
	group.Authorization = auth

	mc.OriginGroups.On("Create", mock.Anything,
		mock.MatchedBy(func(req *origingroups.GroupRequest) bool {
			return req.Authorization != nil &&
				req.Authorization.AuthType == auth.AuthType &&
				req.Authorization.AccessKeyID == auth.AccessKeyID &&
				req.Authorization.AddressingStyle == auth.AddressingStyle &&
				req.Authorization.AwsRegion == auth.AwsRegion &&
				req.Authorization.SecretKey == auth.SecretKey &&
				req.Authorization.BucketName == auth.BucketName
		}),
	).Return(group, nil)

	mc.OriginGroups.On("Get", mock.Anything, int64(testOriginGroupID)).Return(group, nil)

	config := originGroupConfig(testOriginGroupName)
	config["authorization"] = []interface{}{
		map[string]interface{}{
			"auth_type":        auth.AuthType,
			"access_key_id":    auth.AccessKeyID,
			"addressing_style": auth.AddressingStyle,
			"aws_region":       auth.AwsRegion,
			"secret_key":       auth.SecretKey,
			"bucket_name":      auth.BucketName,
		},
	}

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create with s3 authorization",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testOriginGroupID))
			support.RequireStateAttrs(t, state, map[string]string{
				"authorization.#": "1",
			})
		},
	}
}

func originGroupReadCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	drifted := sampleOriginGroup("renamed-out-of-band")
	drifted.UseNext = false
	drifted.ConsistentBalancing = true

	mc.OriginGroups.On("Get", mock.Anything, int64(testOriginGroupID)).Return(drifted, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read overwrites state with API values",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testOriginGroupID),
		CurrentState: originGroupConfig(testOriginGroupName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testOriginGroupID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":                 "renamed-out-of-band",
				"use_next":             "false",
				"consistent_balancing": "true",
				"origin.#":             "1",
			})
		},
	}
}

func originGroupUpdateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	const newName = "tf-origin-group-renamed"

	mc.OriginGroups.On("Update", mock.Anything, int64(testOriginGroupID),
		mock.MatchedBy(func(req *origingroups.GroupRequest) bool {
			return req.Name == newName && len(req.Origins) == 1
		}),
	).Return(sampleOriginGroup(newName), nil)

	mc.OriginGroups.On("Get", mock.Anything, int64(testOriginGroupID)).
		Return(sampleOriginGroup(newName), nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update name",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testOriginGroupID),
		CurrentState: originGroupConfig(testOriginGroupName),
		NewConfig:    originGroupConfig(newName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": newName,
			})
		},
	}
}

func originGroupDeleteCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.OriginGroups.On("Delete", mock.Anything, int64(testOriginGroupID)).Return(nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "delete origin group",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testOriginGroupID),
		CurrentState: originGroupConfig(testOriginGroupName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func originGroupCreateAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.OriginGroups.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("api error: origin group name already taken"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: originGroupConfig(testOriginGroupName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "already taken")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func originGroupDeleteAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.OriginGroups.On("Delete", mock.Anything, int64(testOriginGroupID)).
		Return(fmt.Errorf("api error: origin group is in use"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on delete keeps state",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testOriginGroupID),
		CurrentState: originGroupConfig(testOriginGroupName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "in use")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testOriginGroupID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func originGroupCreateWithBackupCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	group := sampleOriginGroup(testOriginGroupName)
	group.Origins = append(group.Origins, origingroups.Origin{
		ID: 12, Source: "backup.example.com", Enabled: true, Backup: true,
	})

	mc.OriginGroups.On("Create", mock.Anything,
		mock.MatchedBy(func(req *origingroups.GroupRequest) bool {
			if len(req.Origins) != 2 {
				return false
			}

			backups := 0
			for _, o := range req.Origins {
				if o.Backup {
					backups++
				}
			}

			return backups == 1
		}),
	).Return(group, nil)

	mc.OriginGroups.On("Get", mock.Anything, int64(testOriginGroupID)).Return(group, nil)

	config := originGroupConfig(testOriginGroupName)
	config["origin"] = []interface{}{
		map[string]interface{}{"source": testOriginSource, "enabled": true, "backup": false},
		map[string]interface{}{"source": "backup.example.com", "enabled": true, "backup": true},
	}

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create with a backup origin",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"origin.#": "2",
			})
		},
	}
}

func originGroupUpdateDropsAuthCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	withoutAuth := sampleOriginGroup(testOriginGroupName)

	mc.OriginGroups.On("Update", mock.Anything, int64(testOriginGroupID),
		mock.MatchedBy(func(req *origingroups.GroupRequest) bool {
			return req.Authorization == nil
		}),
	).Return(withoutAuth, nil)

	mc.OriginGroups.On("Get", mock.Anything, int64(testOriginGroupID)).Return(withoutAuth, nil)

	current := originGroupConfig(testOriginGroupName)
	current["authorization"] = []interface{}{
		map[string]interface{}{
			"auth_type":        "aws_signature_v4",
			"access_key_id":    "AKIAIOSFODNN7EXAMPLE",
			"addressing_style": "path",
			"aws_region":       "us-east-1",
			"secret_key":       "wJalrXUtnFEMI-K7MDENG-bPxRfiCYEXAMPLEKEY",
			"bucket_name":      "tf-bucket",
		},
	}

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update removing the authorization block sends a null authorization",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testOriginGroupID),
		CurrentState: current,
		NewConfig:    originGroupConfig(testOriginGroupName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"authorization.#": "0",
			})
		},
	}
}

func originGroupUpdateAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.OriginGroups.On("Update", mock.Anything, int64(testOriginGroupID), mock.Anything).
		Return(nil, fmt.Errorf("api error: origin is unreachable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on update",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testOriginGroupID),
		CurrentState: originGroupConfig(testOriginGroupName),
		NewConfig:    originGroupConfig("tf-origin-group-renamed"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "origin is unreachable")
			require.NotNil(t, state, "state must survive a failed update")
		},
	}
}

func originGroupReadAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.OriginGroups.On("Get", mock.Anything, int64(testOriginGroupID)).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on read",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testOriginGroupID),
		CurrentState: originGroupConfig(testOriginGroupName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
		},
	}
}

func TestIntegrationOriginGroup_TableDriven(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_origingroup")

	cases := []support.ResourceCase[*cdnmock.MockedCDN]{
		originGroupCreateCase(),
		originGroupCreateWithAuthCase(),
		originGroupCreateWithBackupCase(),
		originGroupReadCase(),
		originGroupUpdateCase(),
		originGroupUpdateDropsAuthCase(),
		originGroupDeleteCase(),
		originGroupCreateAPIFailureCase(),
		originGroupUpdateAPIFailureCase(),
		originGroupReadAPIFailureCase(),
		originGroupDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cdnmock.MockedCDN])
}
