//go:build integration

package websec_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/websec/lecerts"
	"github.com/Edge-Center/edgecentercdn-go/websec/sslcerts"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	websecmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/websec/mock"
	websecsvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/websec"
)

const (
	testLECertLastRequestID int64 = 5
	testLECertRequestID     int64 = 6

	leCertIssueLockedBody      = `{"le_issue":"You cannot stop the issuing process until the current issuing attempt is in progress. Please wait until the attempt is completed and try again."}`
	leCertIssueDeactivatedBody = `{"errors":{"le_issue":["The certificate issuing process has already been deactivated."]}}`
	leCertIssueRunningBody     = `{"errors":{"resource":["Certificate issuing process have already being launched."]}}`
	leCertRevokeSharedBody     = `{"errors":{"error":["Several resources [42, 43] are linked to this certificate. Use the ?force=true query param to revoke certificate from several resources."]}}`
)

func leCertConfig() map[string]interface{} {
	return map[string]interface{}{"resource_id": int(testResourceID)}
}

func leCertState() map[string]interface{} {
	return map[string]interface{}{"resource_id": int(testResourceID), "cert_type": "LE"}
}

func leCertIssued() *lecerts.Status {
	return &lecerts.Status{
		ID:       testLECertRequestID,
		Resource: testResourceID,
		SSLID:    ptr(testCertID),
		CertType: ptr(lecerts.CertTypeLE),
		Statuses: []lecerts.StatusDetail{{ID: 12, Status: "DONE"}},
	}
}

func leCertFailedAttempt(active bool) *lecerts.Status {
	return &lecerts.Status{
		ID:       testLECertRequestID,
		Resource: testResourceID,
		Active:   active,
		Statuses: []lecerts.StatusDetail{{
			ID:      13,
			Status:  "FAILED",
			Error:   "ValidationIssue",
			Details: "Unable to resolve the domain name - example.com.",
		}},
	}
}

func TestIntegrationWebsecLECert_Create(t *testing.T) {
	t.Parallel()

	resource := websecsvc.ResourceWebsecLECert(time.Millisecond)

	issueRunning := apiError(t, http.StatusBadRequest, cdnsdk.ErrBadRequest, leCertIssueRunningBody)
	unavailable := serverError(t)

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "create waits for a request newer than the one the status shows right after the post",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				previous := &lecerts.Status{
					ID:       testLECertLastRequestID,
					Resource: testResourceID,
					Statuses: []lecerts.StatusDetail{{ID: 11, Status: "DONE"}},
				}
				running := &lecerts.Status{ID: testLECertRequestID, Resource: testResourceID, Active: true}

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(previous, nil).Twice()
				mc.LECerts.On("Issue", mock.Anything, testResourceID, &lecerts.IssueRequest{CertType: lecerts.CertTypeLE}).Return(nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(running, nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(leCertIssued(), nil).Twice()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID), siteSSLAutomated), nil).Once()

				return mc
			},
			NewConfig: leCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateID(t, state, testResourceIDStr)
				support.RequireStateAttrs(t, state, map[string]string{
					"resource_id":     testResourceIDStr,
					"cert_type":       "LE",
					"ssl_certificate": testCertIDStr,
					"issue_status":    "DONE",
					"issue_error":     "",
				})
			},
		},
		{
			Name: "create of the first certificate keeps polling while the status is not found and sends the cert type",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				issued := leCertIssued()
				issued.CertType = ptr(lecerts.CertTypeMDDC)

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Twice()
				mc.LECerts.On("Issue", mock.Anything, testResourceID, &lecerts.IssueRequest{CertType: lecerts.CertTypeMDDC}).Return(nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(issued, nil).Twice()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID), siteSSLAutomated), nil).Once()

				return mc
			},
			NewConfig: map[string]interface{}{"resource_id": int(testResourceID), "cert_type": "MDDC"},
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"cert_type":       "MDDC",
					"ssl_certificate": testCertIDStr,
				})
			},
		},
		{
			Name: "create refuses a resource that uses its own certificate, which the issued one would replace",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID)), nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(&sslcerts.Cert{ID: testCertID}, nil).Once()

				return mc
			},
			NewConfig: leCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "remove ssl_certificate from edgecenter_websec_resource in the same apply")
				require.Nil(t, state)
			},
		},
		{
			Name: "create refuses a resource that already has an automated certificate and points to import",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID)), nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(&sslcerts.Cert{ID: testCertID, Automated: true}, nil).Once()

				return mc
			},
			NewConfig: leCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "terraform import edgecenter_websec_lecert.<name> 42")
				require.Nil(t, state)
			},
		},
		{
			Name: "a rejected issue post leaves no state",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()
				mc.LECerts.On("Issue", mock.Anything, testResourceID, mock.Anything).Return(issueRunning).Once()

				return mc
			},
			NewConfig: leCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "Certificate issuing process have already being launched.")
				require.Nil(t, state)
			},
		},
		{
			Name: "create cancels the request after its first failed attempt and keeps the id for the next apply to clean up",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()
				mc.LECerts.On("Issue", mock.Anything, testResourceID, mock.Anything).Return(nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(leCertFailedAttempt(true), nil).Once()
				mc.LECerts.On("Cancel", mock.Anything, testResourceID).Return(nil).Once()

				return mc
			},
			NewConfig: leCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "ended with FAILED: ValidationIssue: Unable to resolve the domain name - example.com.")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
		{
			Name: "create does not cancel a request that the server has already closed",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				forbidden := &lecerts.Status{
					ID:       testLECertRequestID,
					Resource: testResourceID,
					Statuses: []lecerts.StatusDetail{{
						ID:      13,
						Status:  "FORBIDDEN",
						Details: "Issuing of Let's Encrypt certificates is disabled for your account. Please contact a support team.",
					}},
				}

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()
				mc.LECerts.On("Issue", mock.Anything, testResourceID, mock.Anything).Return(nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(forbidden, nil).Once()

				return mc
			},
			NewConfig: leCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "ended with FORBIDDEN: Issuing of Let's Encrypt certificates is disabled for your account.")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
		{
			Name: "a failed cancel is reported together with the reason of the failed request",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()
				mc.LECerts.On("Issue", mock.Anything, testResourceID, mock.Anything).Return(nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(leCertFailedAttempt(true), nil).Once()
				mc.LECerts.On("Cancel", mock.Anything, testResourceID).Return(unavailable).Once()

				return mc
			},
			NewConfig: leCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "ended with FAILED: ValidationIssue")
				support.RequireErrorDiagContains(t, diags, "cancelling the request failed")
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecLECert_CreateTimeout(t *testing.T) {
	t.Parallel()

	resource := websecsvc.ResourceWebsecLECert(time.Hour)

	var (
		createCtx    context.Context
		cancelCtxErr error
	)

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "create stops polling a margin before its timeout and keeps the id",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).
					Run(func(args mock.Arguments) { createCtx = args.Get(0).(context.Context) }).
					Return(nil, notFound()).Twice()
				mc.LECerts.On("Issue", mock.Anything, testResourceID, mock.Anything).Return(nil).Once()

				return mc
			},
			NewConfig: map[string]interface{}{
				"resource_id": int(testResourceID),
				"timeouts":    map[string]interface{}{"create": "30s"},
			},
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "was accepted but has not started within the create timeout")
				require.ErrorIs(t, createCtx.Err(), context.Canceled, "create must give up before its context runs out")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
		{
			Name: "create tells a request that started from one that never did when its timeout runs out",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()
				mc.LECerts.On("Issue", mock.Anything, testResourceID, mock.Anything).Return(nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).
					Return(&lecerts.Status{ID: testLECertRequestID, Resource: testResourceID, Active: true}, nil).Once()

				return mc
			},
			NewConfig: map[string]interface{}{
				"resource_id": int(testResourceID),
				"timeouts":    map[string]interface{}{"create": "30s"},
			},
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "did not end within the create timeout")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
		{
			Name: "the cancel of a failed request reaches the api although the create timeout has run out",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()
				mc.LECerts.On("Issue", mock.Anything, testResourceID, mock.Anything).Return(nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).
					Run(func(args mock.Arguments) { <-args.Get(0).(context.Context).Done() }).
					Return(leCertFailedAttempt(true), nil).Once()
				mc.LECerts.On("Cancel", mock.Anything, testResourceID).
					Run(func(args mock.Arguments) { cancelCtxErr = args.Get(0).(context.Context).Err() }).
					Return(nil).Once()

				return mc
			},
			NewConfig: map[string]interface{}{
				"resource_id": int(testResourceID),
				"timeouts":    map[string]interface{}{"create": "100ms"},
			},
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "ended with FAILED")
				require.NoError(t, cancelCtxErr)
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecLECert_Read(t *testing.T) {
	t.Parallel()

	resource := websecsvc.ResourceWebsecLECert(time.Millisecond)

	unavailable := serverError(t)

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "read keeps the certificate that the last request issued",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID), siteSSLAutomated), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(leCertIssued(), nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"ssl_certificate": testCertIDStr,
					"issue_status":    "DONE",
					"issue_error":     "",
				})
			},
		},
		{
			Name: "read keeps a request that is still running, so that destroy cancels it, and reports its newest status",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				retried := leCertFailedAttempt(true)
				retried.Statuses = append(retried.Statuses, lecerts.StatusDetail{ID: 9, Status: "FAILED", Error: "InternalIssue"})

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(retried, nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"ssl_certificate": "0",
					"issue_status":    "FAILED",
					"issue_error":     "ValidationIssue: Unable to resolve the domain name - example.com.",
				})
			},
		},
		{
			Name: "read removes the certificate when the last request ended without one",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				cancelled := leCertFailedAttempt(false)
				cancelled.Statuses = append(cancelled.Statuses, lecerts.StatusDetail{ID: 14, Status: "CANCELLED"})

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(cancelled, nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "read keeps an automated certificate that has no request on record",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID)), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(&sslcerts.Cert{ID: testCertID, Automated: true}, nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"ssl_certificate": testCertIDStr,
					"issue_status":    "",
				})
			},
		},
		{
			Name: "read removes the certificate when the resource refers to a deleted certificate and has no request",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID)), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(nil, notFound()).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "read removes the certificate when the resource is gone",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "read keeps the state when the api fails for another reason",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID), siteSSLAutomated), nil).Once()
				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, unavailable).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecLECert_Delete(t *testing.T) {
	t.Parallel()

	resource := websecsvc.ResourceWebsecLECert(time.Millisecond)

	locked := apiError(t, http.StatusBadRequest, cdnsdk.ErrBadRequest, leCertIssueLockedBody)
	deactivated := apiError(t, http.StatusBadRequest, cdnsdk.ErrBadRequest, leCertIssueDeactivatedBody)
	revokeShared := apiError(t, http.StatusBadRequest, cdnsdk.ErrBadRequest, leCertRevokeSharedBody)

	running := &lecerts.Status{ID: testLECertRequestID, Resource: testResourceID, Active: true}

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "delete revokes the automated certificate without forcing it",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(leCertIssued(), nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID), siteSSLAutomated), nil).Once()
				mc.LECerts.On("Revoke", mock.Anything, testResourceID, false).Return(nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete revokes a moved certificate that has no request on record",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID)), nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(&sslcerts.Cert{ID: testCertID, Automated: true}, nil).Once()
				mc.LECerts.On("Revoke", mock.Anything, testResourceID, false).Return(nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete cancels a running renewal and revokes the certificate that the cancel leaves attached",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				renewal := &lecerts.Status{ID: testLECertRequestID, Resource: testResourceID, Active: true, SSLID: ptr(testCertID)}

				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(renewal, nil).Once()
				mc.LECerts.On("Cancel", mock.Anything, testResourceID).Return(nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID)), nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(&sslcerts.Cert{ID: testCertID, Automated: true}, nil).Once()
				mc.LECerts.On("Revoke", mock.Anything, testResourceID, false).Return(nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete retries the cancel while an issue attempt holds the request",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(running, nil).Twice()
				mc.LECerts.On("Cancel", mock.Anything, testResourceID).Return(locked).Once()
				mc.LECerts.On("Cancel", mock.Anything, testResourceID).Return(nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete does not retry a cancel that fails for another reason",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(running, nil).Once()
				mc.LECerts.On("Cancel", mock.Anything, testResourceID).Return(deactivated).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "The certificate issuing process has already been deactivated.")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
		{
			Name: "delete reports a revoke refused for a certificate shared with another resource instead of forcing it",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(leCertIssued(), nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID), siteSSLAutomated), nil).Once()
				mc.LECerts.On("Revoke", mock.Anything, testResourceID, false).Return(revokeShared).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "Several resources [42, 43] are linked to this certificate.")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
		{
			Name: "delete succeeds when the resource is already gone",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()

				mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: leCertState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecLECert_DeleteTimeout(t *testing.T) {
	t.Parallel()

	resource := websecsvc.ResourceWebsecLECert(time.Hour)

	t.Run("delete stops retrying the cancel a margin before its timeout and keeps the id", func(t *testing.T) {
		t.Parallel()

		mc := websecmock.NewMockedWebsec()
		t.Cleanup(func() { mc.MockCleanup(t) })

		running := &lecerts.Status{ID: testLECertRequestID, Resource: testResourceID, Active: true}
		mc.LECerts.On("Get", mock.Anything, testResourceID).Return(running, nil).Once()
		mc.LECerts.On("Cancel", mock.Anything, testResourceID).
			Return(apiError(t, http.StatusBadRequest, cdnsdk.ErrBadRequest, leCertIssueLockedBody)).Once()

		data := resource.Data(support.NewState(t, resource, leCertState(), testResourceIDStr))

		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()

		diags := resource.DeleteContext(ctx, data, mc.TestMeta())

		support.RequireOnlyErrorDiags(t, diags)
		support.RequireErrorDiagContains(t, diags, "You cannot stop the issuing process until the current issuing attempt is in progress.")
		require.NoError(t, ctx.Err(), "delete must give up before its context runs out")
		support.RequireStateID(t, data.State(), testResourceIDStr)
	})
}

func TestIntegrationWebsecLECert_Import(t *testing.T) {
	t.Parallel()

	resource := websecsvc.ResourceWebsecLECert(time.Millisecond)

	t.Run("the id is the resource id", func(t *testing.T) {
		t.Parallel()

		imported, err := runImport(t, resource, websecmock.NewMockedWebsec(), testResourceIDStr)
		require.NoError(t, err)
		require.Equal(t, testResourceIDStr, imported.Id())
		require.Equal(t, int(testResourceID), imported.Get("resource_id"))
	})

	t.Run("a malformed id is rejected with the expected format", func(t *testing.T) {
		t.Parallel()

		for _, id := range []string{"", "abc", "0", "-42", "42:LE", "resource:42"} {
			_, err := runImport(t, resource, websecmock.NewMockedWebsec(), id)
			require.ErrorContainsf(t, err, "must be <resource_id>", "import id %q", id)
		}
	})
}
