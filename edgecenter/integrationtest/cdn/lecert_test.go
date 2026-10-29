//go:build integration

package cdn_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
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

func leCertCreateTakesOverCancelledIssuanceCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: false, Statuses: testLECertStatus("CANCELLED")}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create with active false takes over an already cancelled issuance",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: leCertConfig(false, false),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, "0")
			support.RequireStateAttrs(t, state, map[string]string{
				"active": "false",
				"update": "false",
			})
			mc.LECerts.AssertNotCalled(t, "IssueLECert", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func leCertCreateInactiveRejectsRunningIssuanceCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: true, Statuses: testLECertStatus("FAILED")}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create with active false is rejected while issuance is not cancelled",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: leCertConfig(false, false),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "'active' cannot be set to false when creating")
			support.RequireErrorDiagContains(t, diags, "is not cancelled")
			require.Nil(t, state, "state must be nil when create is rejected")
			mc.LECerts.AssertNotCalled(t, "IssueLECert", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func leCertCreateInactiveRejectsMissingIssuanceCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(nil, fmt.Errorf("request: %w", cdnsdk.NewAPIError(404, cdnsdk.ErrNotFound)))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create with active false is rejected when there is no issuance",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: leCertConfig(false, false),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "'active' cannot be set to false when creating")
			support.RequireErrorDiagContains(t, diags, "has no ACME issuance to take over")
			require.Nil(t, state, "state must be nil when create is rejected")
		},
	}
}

func leCertCreateInactiveRejectsIssuedCertCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: testLECertID, Active: false}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create with active false is rejected when a cert is already issued",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: leCertConfig(false, false),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "already has an issued ACME certificate")
			require.Nil(t, state, "state must be nil when create is rejected")
		},
	}
}

func leCertCreateInactiveReadFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create with active false reports a failed status read",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: leCertConfig(false, false),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, fmt.Sprintf("failed to read ACME cert for resource %d", testLECertResourceID))
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func leCertCreateMDDCCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("IssueLECert", mock.Anything, int64(testLECertResourceID), &lecerts.IssueRequest{CertType: lecerts.CertTypeMDDC}).Return(nil)

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{
			ID:       testLECertID,
			Active:   true,
			Resource: testLECertResourceID,
			Started:  "2024-01-01T00:00:00Z",
			Statuses: testLECertStatus("done"),
		}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:    "create MDDC cert sends cert_type and keeps it in state",
		Op:      support.OpApply,
		Prepare: func() *cdnmock.MockedCDN { return mc },
		NewConfig: map[string]interface{}{
			"resource_id": testLECertResourceID,
			"cert_type":   string(lecerts.CertTypeMDDC),
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testLECertID))
			support.RequireStateAttrs(t, state, map[string]string{
				"cert_type": "MDDC",
				"active":    "true",
				"update":    "false",
			})
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

func leCertReadKeepsCancelledIssuanceCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: false, Statuses: testLECertStatus("CANCELLED")}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read keeps an issuance cancelled through active false",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "0",
		CurrentState: leCertConfig(false, false),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, "0")
			support.RequireStateAttrs(t, state, map[string]string{"active": "false"})
		},
	}
}

func leCertReadKeepsIssuedCertCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: testLECertID, Active: false, Statuses: testLECertStatus("CANCELLED")}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read keeps an issued cert when its renewal is not active",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testLECertID))
			support.RequireStateAttrs(t, state, map[string]string{"active": "true"})
		},
	}
}

func leCertReadPicksUpRestartedIssuanceCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: true, Statuses: testLECertStatus("FAILED")}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read reports active when issuance was started again outside terraform",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "0",
		CurrentState: leCertConfig(false, false),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, "0")
			support.RequireStateAttrs(t, state, map[string]string{"active": "true"})
		},
	}
}

func leCertReadLegacyStateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{
			ID:       testLECertID,
			Active:   true,
			Resource: testLECertResourceID,
			Statuses: testLECertStatus("issued"),
		}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read normalizes missing cert_type to LE",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"cert_type": "LE",
			})
		},
	}
}

func leCertReadCertTypeFromAPICase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{
			ID:       testLECertID,
			Active:   true,
			Resource: testLECertResourceID,
			CertType: lecerts.CertTypeMDDC,
			Statuses: testLECertStatus("issued"),
		}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read takes cert_type from the API when present",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"cert_type": "MDDC",
			})
		},
	}
}

func leCertReadNotFoundClearsStateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(nil, fmt.Errorf("request: %w", cdnsdk.NewAPIError(404, cdnsdk.ErrNotFound)))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read clears state when the API returns 404",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after read gets 404")
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
				"active": "false",
				"update": "false",
			})
		},
	}
}

func leCertUpdateRestartsIssuanceCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: false, Statuses: testLECertStatus("CANCELLED")}, nil)

	mc.Resources.On("Get", mock.Anything, int64(testLECertResourceID)).
		Return(&resources.Resource{ID: testLECertResourceID, SSLData: 0}, nil)

	mc.LECerts.On("IssueLECert", mock.Anything, int64(testLECertResourceID), (*lecerts.IssueRequest)(nil)).Return(nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update restarts issuance when active is enabled again",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "0",
		CurrentState: leCertConfig(false, false),
		NewConfig:    leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{"active": "true"})
			mc.LECerts.AssertNotCalled(t, "CancelLECert", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func leCertUpdateRestartsMDDCIssuanceCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: false, Statuses: testLECertStatus("CANCELLED")}, nil)

	mc.Resources.On("Get", mock.Anything, int64(testLECertResourceID)).
		Return(&resources.Resource{ID: testLECertResourceID, SSLData: 0}, nil)

	mc.LECerts.On("IssueLECert", mock.Anything, int64(testLECertResourceID), &lecerts.IssueRequest{CertType: lecerts.CertTypeMDDC}).Return(nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "update restarts MDDC issuance with its cert_type",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		CurrentID: "0",
		CurrentState: map[string]interface{}{
			"resource_id": testLECertResourceID,
			"cert_type":   string(lecerts.CertTypeMDDC),
			"update":      false,
			"active":      false,
		},
		NewConfig: map[string]interface{}{
			"resource_id": testLECertResourceID,
			"cert_type":   string(lecerts.CertTypeMDDC),
			"active":      true,
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"active":    "true",
				"cert_type": "MDDC",
			})
		},
	}
}

func leCertUpdateRenewKeepsInactiveCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: testLECertID, Active: true, Resource: testLECertResourceID}, nil)

	mc.Resources.On("Get", mock.Anything, int64(testLECertResourceID)).
		Return(&resources.Resource{ID: testLECertResourceID, SSLData: testLECertID}, nil)

	mc.LECerts.On("UpdateLECert", mock.Anything, int64(testLECertResourceID)).Return(nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update renews without cancelling when active stays false",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, false),
		NewConfig:    leCertConfig(true, false),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			mc.LECerts.AssertCalled(t, "UpdateLECert", mock.Anything, int64(testLECertResourceID))
		},
	}
}

func leCertUpdateCancelFailureKeepsActiveCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: true, Statuses: testLECertStatus("PROCESSING")}, nil)

	mc.Resources.On("Get", mock.Anything, int64(testLECertResourceID)).
		Return(&resources.Resource{ID: testLECertResourceID, SSLData: 0}, nil)

	mc.LECerts.On("CancelLECert", mock.Anything, int64(testLECertResourceID), false).
		Return(fmt.Errorf("request: %w", cdnsdk.NewAPIError(400, cdnsdk.ErrBadRequest)))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "failed cancel leaves active true in state",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "0",
		CurrentState: leCertConfig(false, true),
		NewConfig:    leCertConfig(false, false),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "failed to cancel ACME cert")
			support.RequireStateAttrs(t, state, map[string]string{"active": "true"})
		},
	}
}

func leCertUpdateRestartFailureKeepsInactiveCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: false, Statuses: testLECertStatus("CANCELLED")}, nil)

	mc.Resources.On("Get", mock.Anything, int64(testLECertResourceID)).
		Return(&resources.Resource{ID: testLECertResourceID, SSLData: 0}, nil)

	mc.LECerts.On("IssueLECert", mock.Anything, int64(testLECertResourceID), (*lecerts.IssueRequest)(nil)).
		Return(fmt.Errorf("request: %w", cdnsdk.NewAPIError(400, cdnsdk.ErrBadRequest)))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "failed restart leaves active false in state",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "0",
		CurrentState: leCertConfig(false, false),
		NewConfig:    leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "failed to restart ACME cert issuance")
			support.RequireStateAttrs(t, state, map[string]string{"active": "false"})
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

func leCertDeletePendingCancelsIssuanceCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("DeleteLECert", mock.Anything, int64(testLECertResourceID), true).
		Return(fmt.Errorf("request: %w", cdnsdk.NewAPIError(400, cdnsdk.ErrBadRequest)))

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: true}, nil)

	mc.LECerts.On("CancelLECert", mock.Anything, int64(testLECertResourceID), false).Return(nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "delete cancels pending issuance when revoke has nothing to revoke",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "0",
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete cancels the issuance")
		},
	}
}

func leCertDeleteAlreadyCancelledCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("DeleteLECert", mock.Anything, int64(testLECertResourceID), true).
		Return(fmt.Errorf("request: %w", cdnsdk.NewAPIError(400, cdnsdk.ErrBadRequest)))

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: false}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "delete succeeds when issuance is already cancelled",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "0",
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete of a cancelled issuance")
			mc.LECerts.AssertNotCalled(t, "CancelLECert", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func leCertDeleteNotFoundCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("DeleteLECert", mock.Anything, int64(testLECertResourceID), true).
		Return(fmt.Errorf("request: %w", cdnsdk.NewAPIError(404, cdnsdk.ErrNotFound)))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "delete succeeds when the cert is already gone",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete of a missing cert")
		},
	}
}

func leCertDeleteRevokeFailsOnIssuedCertCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("DeleteLECert", mock.Anything, int64(testLECertResourceID), true).
		Return(fmt.Errorf("request: %w", cdnsdk.NewAPIError(400, cdnsdk.ErrBadRequest)))

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: testLECertID, Active: true}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "delete keeps state when revoke fails for an issued cert",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testLECertID),
		CurrentState: leCertConfig(false, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when revoke genuinely fails")
		},
	}
}

func leCertUpdateRenewNotIssuedCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).
		Return(&lecerts.LECertStatus{ID: 0, Active: true}, nil)

	mc.Resources.On("Get", mock.Anything, int64(testLECertResourceID)).
		Return(&resources.Resource{ID: testLECertResourceID, SSLData: 0}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update rejects renew when the cert is not issued yet",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "0",
		CurrentState: leCertConfig(false, true),
		NewConfig:    leCertConfig(true, true),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "not issued yet")
			mc.LECerts.AssertNotCalled(t, "UpdateLECert", mock.Anything, mock.Anything)
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
		leCertCreateTakesOverCancelledIssuanceCase(),
		leCertCreateInactiveRejectsRunningIssuanceCase(),
		leCertCreateInactiveRejectsMissingIssuanceCase(),
		leCertCreateInactiveRejectsIssuedCertCase(),
		leCertCreateInactiveReadFailureCase(),
		leCertCreateMDDCCase(),
		leCertReadCase(),
		leCertReadAbsentCase(),
		leCertReadKeepsCancelledIssuanceCase(),
		leCertReadKeepsIssuedCertCase(),
		leCertReadPicksUpRestartedIssuanceCase(),
		leCertReadLegacyStateCase(),
		leCertReadCertTypeFromAPICase(),
		leCertReadNotFoundClearsStateCase(),
		leCertUpdateReissueCase(),
		leCertUpdateSkipsReissueOnMismatchCase(),
		leCertUpdateCancelCase(),
		leCertUpdateRestartsIssuanceCase(),
		leCertUpdateRestartsMDDCIssuanceCase(),
		leCertUpdateRenewKeepsInactiveCase(),
		leCertUpdateCancelFailureKeepsActiveCase(),
		leCertUpdateRestartFailureKeepsInactiveCase(),
		leCertUpdateRenewNotIssuedCase(),
		leCertDeleteCase(),
		leCertDeletePendingCancelsIssuanceCase(),
		leCertDeleteAlreadyCancelledCase(),
		leCertDeleteNotFoundCase(),
		leCertDeleteRevokeFailsOnIssuedCertCase(),
		leCertCreateAPIFailureCase(),
		leCertUpdateAPIFailureCase(),
		leCertReadAPIFailureCase(),
		leCertDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cdnmock.MockedCDN])
}

func TestIntegrationLECert_DiffGuards(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_lecert")

	cases := []struct {
		name       string
		id         string
		state      map[string]interface{}
		config     map[string]interface{}
		wantErr    string
		wantForced bool
	}{
		{
			name:    "create with update=true is rejected",
			config:  map[string]interface{}{"resource_id": testLECertResourceID, "update": true},
			wantErr: "'update' cannot be set to true when creating",
		},
		{
			name:   "cert_type change on existing cert is rejected",
			id:     fmt.Sprintf("%d", testLECertID),
			state:  map[string]interface{}{"resource_id": testLECertResourceID, "cert_type": "LE"},
			config: map[string]interface{}{"resource_id": testLECertResourceID, "cert_type": "MDDC"},

			wantErr: "cert_type cannot be changed",
		},
		{
			name:   "legacy state without cert_type plans clean",
			id:     fmt.Sprintf("%d", testLECertID),
			state:  map[string]interface{}{"resource_id": testLECertResourceID},
			config: map[string]interface{}{"resource_id": testLECertResourceID},
		},
		{
			name:    "moving to another resource with active=false is rejected",
			id:      fmt.Sprintf("%d", testLECertID),
			state:   map[string]interface{}{"resource_id": testLECertResourceID, "cert_type": "LE", "active": true},
			config:  map[string]interface{}{"resource_id": testLECertResourceID + 1, "active": false},
			wantErr: "'active' cannot be set to false when moving the certificate to another resource",
		},
		{
			name:    "moving an inactive cert to another resource is rejected",
			id:      "0",
			state:   map[string]interface{}{"resource_id": testLECertResourceID, "cert_type": "LE", "active": false},
			config:  map[string]interface{}{"resource_id": testLECertResourceID + 1, "active": false},
			wantErr: "'active' cannot be set to false when moving the certificate to another resource",
		},
		{
			name:       "resource_id change forces replacement",
			id:         fmt.Sprintf("%d", testLECertID),
			state:      map[string]interface{}{"resource_id": testLECertResourceID, "cert_type": "LE"},
			config:     map[string]interface{}{"resource_id": testLECertResourceID + 1},
			wantForced: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var current *terraform.InstanceState
			if tc.id != "" {
				current = support.NewState(t, resource, tc.state, tc.id)
			}

			diff, err := resource.Diff(context.Background(), current, terraform.NewResourceConfigRaw(tc.config), nil)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			if tc.wantForced {
				require.NotNil(t, diff)
				require.True(t, diff.RequiresNew(), "resource_id change must force replacement")

				return
			}
			if diff != nil {
				require.False(t, diff.RequiresNew(), "legacy state must not trigger replacement")
			}
		})
	}
}

func TestIntegrationLECert_ImportID(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_lecert")

	cases := []struct {
		name         string
		importID     string
		status       *lecerts.LECertStatus
		statusErr    error
		wantErr      string
		wantResource string
		wantCertType string
	}{
		{
			name:         "plain resource id",
			importID:     "663775",
			status:       &lecerts.LECertStatus{ID: testLECertID, Active: true, CertType: lecerts.CertTypeLE},
			wantResource: "663775",
		},
		{
			name:     "plain resource id is rejected while issuance is cancelled",
			importID: "663775",
			status:   &lecerts.LECertStatus{ID: 0, Active: false, Statuses: testLECertStatus("CANCELLED")},
			wantErr:  "import it as 663775:LE or 663775:MDDC",
		},
		{
			name:     "plain resource id is rejected while issuance is not finished",
			importID: "663775",
			status:   &lecerts.LECertStatus{ID: 0, Active: true, Statuses: testLECertStatus("FAILED")},
			wantErr:  "import it as 663775:LE or 663775:MDDC",
		},
		{
			name:         "plain resource id is left to read when the status is missing",
			importID:     "663775",
			statusErr:    fmt.Errorf("request: %w", cdnsdk.NewAPIError(404, cdnsdk.ErrNotFound)),
			wantResource: "663775",
		},
		{
			name:         "resource id with cert type",
			importID:     "663775:MDDC",
			wantResource: "663775",
			wantCertType: "MDDC",
		},
		{
			name:     "non-numeric resource id",
			importID: "abc",
			wantErr:  "invalid import id",
		},
		{
			name:     "unknown cert type",
			importID: "663775:FOO",
			wantErr:  "invalid cert_type",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mc := cdnmock.NewMockedCDN()
			if tc.status != nil || tc.statusErr != nil {
				mc.LECerts.On("GetLECert", mock.Anything, int64(663775)).Return(tc.status, tc.statusErr)
			}

			data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{})
			data.SetId(tc.importID)

			results, err := resource.Importer.StateContext(context.Background(), data, mc.TestMeta())

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Len(t, results, 1)
			require.Equal(t, tc.importID, results[0].Id())
			require.Equal(t, tc.wantResource, fmt.Sprintf("%d", results[0].Get("resource_id").(int)))
			if tc.wantCertType != "" {
				require.Equal(t, tc.wantCertType, results[0].Get("cert_type").(string))
			}
		})
	}
}

func TestIntegrationLECert_ImportThenRead(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_lecert")

	cases := []struct {
		name      string
		importID  string
		status    *lecerts.LECertStatus
		wantID    string
		wantAttrs map[string]string
	}{
		{
			name:      "idle issuance imported with cert type stays inactive",
			importID:  fmt.Sprintf("%d:MDDC", testLECertResourceID),
			status:    &lecerts.LECertStatus{ID: 0, Active: false, Statuses: testLECertStatus("CANCELLED")},
			wantID:    "0",
			wantAttrs: map[string]string{"active": "false", "cert_type": "MDDC"},
		},
		{
			name:      "issued cert imported by resource id takes cert_type from the API",
			importID:  fmt.Sprintf("%d", testLECertResourceID),
			status:    &lecerts.LECertStatus{ID: testLECertID, Active: true, CertType: lecerts.CertTypeMDDC},
			wantID:    fmt.Sprintf("%d", testLECertID),
			wantAttrs: map[string]string{"active": "true", "cert_type": "MDDC"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mc := cdnmock.NewMockedCDN()
			mc.LECerts.On("GetLECert", mock.Anything, int64(testLECertResourceID)).Return(tc.status, nil)

			data := resource.Data(nil)
			data.SetId(tc.importID)

			results, err := resource.Importer.StateContext(context.Background(), data, mc.TestMeta())
			require.NoError(t, err)
			require.Len(t, results, 1)

			state, diags := resource.RefreshWithoutUpgrade(context.Background(), results[0].State(), mc.TestMeta())
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, tc.wantID)
			support.RequireStateAttrs(t, state, tc.wantAttrs)
		})
	}
}
