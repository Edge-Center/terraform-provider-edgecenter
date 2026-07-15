//go:build integration

package cdn_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercdn-go/lecerts"
	"github.com/Edge-Center/edgecentercdn-go/resources"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	cdnmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cdn/mock"
)

const (
	testLECertResourceID = 77
	testLECertID         = 4242
)

func testLECertStatus(status string) []lecerts.LEStatusDetail {
	return []lecerts.LEStatusDetail{
		{
			Status:  status,
			Error:   "",
			Details: "validation completed",
			Created: "2024-01-01T00:00:00Z",
		},
	}
}

func leCertConfig(update, active bool) map[string]interface{} {
	return map[string]interface{}{
		"resource_id": testLECertResourceID,
		"update":      update,
		"active":      active,
	}
}

func leCertCreateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("IssueLECert", mock.Anything, int64(testLECertResourceID), (*lecerts.IssueRequest)(nil)).Return(nil)

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{
			ID:       testLECertID,
			Active:   true,
			Resource: testLECertResourceID,
			Started:  "2024-01-01T00:00:00Z",
			Statuses: testLECertStatus("done"),
		}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testLECertID))
			support.RequireStateAttrs(t, state, map[string]string{
				"resource_id": fmt.Sprintf("%d", testLECertResourceID),
				"active":      "true",
				"update":      "false",
			})
		},
	}
}

func leCertCreateInactiveCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("IssueLECert", mock.Anything, int64(testLECertResourceID), (*lecerts.IssueRequest)(nil)).Return(nil)

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: false}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create clears state when cert is absent and inactive",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil when read clears an absent cert")
		},
	}
}

func leCertReadCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{
			ID:       testLECertID,
			Active:   true,
			Resource: testLECertResourceID,
			Started:  "2024-01-01T00:00:00Z",
			Statuses: testLECertStatus("issued"),
		}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read takes the id from the API",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "1",
		CurrentState: leCertConfig(true, false),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testLECertID))
			support.RequireStateAttrs(t, state, map[string]string{
				"active": "true",
				"update": "false",
			})
		},
	}
}

func leCertReadAbsentCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: false}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read clears state when cert is absent and inactive",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after read clears an absent cert")
		},
	}
}

func leCertUpdateReissueCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{
			ID:       testLECertID,
			Active:   true,
			Resource: testLECertResourceID,
			Statuses: testLECertStatus("renewing"),
		}, nil)

	mc.Resources.On("Get", mock.Anything, int64(testLECertResourceID)).
		Return(&resources.Resource{ID: testLECertResourceID, SSLData: testLECertID}, nil)

	mc.LECerts.On("UpdateLECert", mock.Anything, int64(testLECertResourceID)).Return(nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update reissues cert when update flag is set and ssl_data matches",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		NewConfig:    leCertConfig(true, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"update": "false",
				"active": "true",
			})
		},
	}
}

func leCertUpdateCancelCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{
			ID:       testLECertID,
			Active:   true,
			Resource: testLECertResourceID,
			Statuses: testLECertStatus("cancelled"),
		}, nil)

	mc.Resources.On("Get", mock.Anything, int64(testLECertResourceID)).
		Return(&resources.Resource{ID: testLECertResourceID, SSLData: testLECertID}, nil)

	mc.LECerts.On("CancelLECert", mock.Anything, int64(testLECertResourceID), false).Return(nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update cancels issuance when active is disabled",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		NewConfig:    leCertConfig(false, false),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"active": "true",
				"update": "false",
			})
		},
	}
}

func leCertDeleteCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("DeleteLECert", mock.Anything, int64(testLECertResourceID), true).Return(nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "delete cert",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func leCertCreateAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("IssueLECert", mock.Anything, int64(testLECertResourceID), (*lecerts.IssueRequest)(nil)).
		Return(fmt.Errorf("api error: domain validation failed"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "domain validation failed")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func leCertDeleteAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("DeleteLECert", mock.Anything, int64(testLECertResourceID), true).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on delete keeps state",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testLECertID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

// The reissue branch is guarded by cert.ID == resource.SSLData: when the resource points
// at a different certificate, UpdateLECert must not be called at all.
func leCertUpdateSkipsReissueOnMismatchCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: testLECertID, Active: true}, nil)

	mc.Resources.On("Get", mock.Anything, int64(testLECertResourceID)).
		Return(&resources.Resource{ID: testLECertResourceID, SSLData: 9999}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update does not reissue when ssl_data points elsewhere",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		NewConfig:    leCertConfig(true, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			mc.LECerts.AssertNotCalled(t, "UpdateLECert", mock.Anything, mock.Anything)
		},
	}
}

func leCertUpdateAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on update",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		NewConfig:    leCertConfig(true, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
		},
	}
}

func leCertReadAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on read",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
		},
	}
}

func TestIntegrationLECert_TableDriven(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_lecert")

	cases := []support.ResourceCase[*cdnmock.MockedCDN]{
		leCertCreateCase(),
		leCertCreateInactiveCase(),
		leCertReadCase(),
		leCertReadAbsentCase(),
		leCertUpdateReissueCase(),
		leCertUpdateSkipsReissueOnMismatchCase(),
		leCertUpdateCancelCase(),
		leCertDeleteCase(),
		leCertCreateAPIFailureCase(),
		leCertUpdateAPIFailureCase(),
		leCertReadAPIFailureCase(),
		leCertDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cdnmock.MockedCDN])
}
