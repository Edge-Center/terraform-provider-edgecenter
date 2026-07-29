//go:build integration

package dns_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	dnssdk "github.com/Edge-Center/edgecenter-dns-sdk-go"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	dnsmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/dns/mock"
)

const (
	secondaryZoneName       = "secondary.example.com"
	secondaryPaddedID       = "  secondary.example.com  "
	secondaryRenamedName    = "renamed.secondary.example.com"
	secondaryPaddedName     = "  padded.secondary.example.com  "
	secondaryTrimmedName    = "padded.secondary.example.com"
	secondaryZoneID         = 4242
	secondaryStaleZoneID    = 7
	secondaryStaleUpdatedAt = "1999-01-01T00:00:00Z"
	secondaryMaster         = "192.0.2.10"
	secondaryPaddedMaster   = "  192.0.2.10  "
	secondaryTSIGKey        = "c2VjcmV0LXRzaWcta2V5"
	secondaryTSIGName       = "transfer-key.secondary.example.com"
	secondaryUpdatedAtNS    = 1700000000123456789
	secondaryOtherMaster    = "198.51.100.20"
	secondaryOtherTSIGKey   = "b3RoZXItdHNpZy1rZXk="
	secondaryOtherTSIG      = "adopted-key.secondary.example.com"
)

func secondaryConfig(master string) map[string]interface{} {
	return map[string]interface{}{
		"name":   secondaryZoneName,
		"master": master,
	}
}

func secondaryTSIGConfig(master, key, tsigName string) map[string]interface{} {
	config := secondaryConfig(master)
	config["tsig_key"] = key
	config["tsig_name"] = tsigName

	return config
}

func secondaryStaleComputedState() map[string]interface{} {
	state := secondaryTSIGConfig(secondaryMaster, secondaryTSIGKey, secondaryTSIGName)
	state["zone_id"] = secondaryStaleZoneID
	state["updated_at"] = secondaryStaleUpdatedAt

	return state
}

func secondaryAPIZone(tsig *dnssdk.TsigOptions, updatedAt dnssdk.Timestamp) dnssdk.SecondaryZone {
	return secondaryAPIZoneNamed(secondaryZoneName, tsig, updatedAt)
}

func secondaryAPIZoneNamed(name string, tsig *dnssdk.TsigOptions, updatedAt dnssdk.Timestamp) dnssdk.SecondaryZone {
	return dnssdk.SecondaryZone{
		ID:        secondaryZoneID,
		Name:      name,
		TSIG:      tsig,
		UpdatedAt: updatedAt,
	}
}

func secondaryAPIError(statusCode int, message string) error {
	return dnssdk.APIError{StatusCode: statusCode, Message: message}
}

func secondaryUpdatedAt(nanos int64) string {
	return time.Unix(0, nanos).Format(time.RFC3339)
}

func secondaryCreateCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(dnssdk.SecondaryZone{}, secondaryAPIError(http.StatusNotFound, "zone not found"))

	mc.Client.On("CreateSecondaryZone", mock.Anything,
		mock.MatchedBy(func(req dnssdk.CreateSecondaryZoneRequest) bool {
			return req.Name == secondaryZoneName &&
				req.Master == secondaryMaster &&
				req.Key == secondaryTSIGKey &&
				req.TSIGName == secondaryTSIGName
		}),
	).Return(secondaryAPIZone(
		&dnssdk.TsigOptions{Key: secondaryTSIGKey, Master: secondaryMaster, Name: secondaryTSIGName},
		secondaryUpdatedAtNS,
	), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "create after a 404 lookup",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: secondaryTSIGConfig(secondaryMaster, secondaryTSIGKey, secondaryTSIGName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, secondaryZoneName)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":       secondaryZoneName,
				"master":     secondaryMaster,
				"tsig_name":  secondaryTSIGName,
				"zone_id":    fmt.Sprintf("%d", secondaryZoneID),
				"updated_at": secondaryUpdatedAt(secondaryUpdatedAtNS),
			})
		},
	}
}

// Only the request is trimmed: the padded master survives in state because the
// created zone carries no TSIG to overwrite it with.
func secondaryCreateTrimsNameAndMasterCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryTrimmedName).
		Return(dnssdk.SecondaryZone{}, secondaryAPIError(http.StatusNotFound, "zone not found"))

	mc.Client.On("CreateSecondaryZone", mock.Anything,
		mock.MatchedBy(func(req dnssdk.CreateSecondaryZoneRequest) bool {
			return req.Name == secondaryTrimmedName && req.Master == secondaryMaster
		}),
	).Return(secondaryAPIZoneNamed(secondaryTrimmedName, nil, 0), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:    "create trims surrounding spaces from name and master",
		Op:      support.OpApply,
		Prepare: func() *dnsmock.MockedDNS { return mc },
		NewConfig: map[string]interface{}{
			"name":   secondaryPaddedName,
			"master": secondaryPaddedMaster,
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, secondaryTrimmedName)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":   secondaryTrimmedName,
				"master": secondaryPaddedMaster,
			})
		},
	}
}

func secondaryCreateWithoutTSIGCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(dnssdk.SecondaryZone{}, secondaryAPIError(http.StatusNotFound, "zone not found"))

	mc.Client.On("CreateSecondaryZone", mock.Anything,
		mock.MatchedBy(func(req dnssdk.CreateSecondaryZoneRequest) bool {
			return req.Name == secondaryZoneName &&
				req.Master == secondaryMaster &&
				req.Key == "" &&
				req.TSIGName == ""
		}),
	).Return(secondaryAPIZone(nil, 0), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "create without tsig sends no key and no tsig name",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: secondaryConfig(secondaryMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, secondaryZoneName)
			support.RequireStateAttrs(t, state, map[string]string{
				"master":     secondaryMaster,
				"tsig_name":  "",
				"zone_id":    fmt.Sprintf("%d", secondaryZoneID),
				"updated_at": "",
			})
		},
	}
}

func secondaryCreateAdoptsExistingCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(secondaryAPIZone(
			&dnssdk.TsigOptions{Master: secondaryOtherMaster, Name: secondaryOtherTSIG},
			secondaryUpdatedAtNS,
		), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "create adopts an existing zone instead of creating it",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: secondaryTSIGConfig(secondaryMaster, secondaryTSIGKey, secondaryTSIGName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, secondaryZoneName)
			support.RequireStateAttrs(t, state, map[string]string{
				"master":     secondaryOtherMaster,
				"tsig_name":  secondaryOtherTSIG,
				"tsig_key":   secondaryTSIGKey,
				"zone_id":    fmt.Sprintf("%d", secondaryZoneID),
				"updated_at": secondaryUpdatedAt(secondaryUpdatedAtNS),
			})
			require.Len(t, fake.Client.Calls, 1, "the lookup must be the only client call")
		},
	}
}

func secondaryCreateLookupFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(dnssdk.SecondaryZone{}, secondaryAPIError(http.StatusInternalServerError, "backend down"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "non 404 lookup error aborts create",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: secondaryConfig(secondaryMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "check existing secondary zone")
			support.RequireErrorDiagContains(t, diags, "backend down")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

// errors.As is given a *dnssdk.APIError target, so only a value APIError matches;
// a *dnssdk.APIError with 404 falls through to the generic failure branch.
func secondaryCreatePointerNotFoundCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(dnssdk.SecondaryZone{}, &dnssdk.APIError{StatusCode: http.StatusNotFound, Message: "zone not found"})

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "pointer APIError 404 is not recognized on create",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: secondaryConfig(secondaryMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "check existing secondary zone")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func secondaryCreateAPIFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(dnssdk.SecondaryZone{}, secondaryAPIError(http.StatusNotFound, "zone not found"))

	mc.Client.On("CreateSecondaryZone", mock.Anything, mock.Anything).
		Return(dnssdk.SecondaryZone{}, fmt.Errorf("api error: secondary zone quota exceeded"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: secondaryConfig(secondaryMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "create secondary zone")
			support.RequireErrorDiagContains(t, diags, "quota exceeded")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func secondaryReadCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(secondaryAPIZone(
			&dnssdk.TsigOptions{Master: secondaryOtherMaster, Name: secondaryOtherTSIG},
			secondaryUpdatedAtNS,
		), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read overwrites master and tsig_name with API values",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryTSIGConfig(secondaryMaster, secondaryTSIGKey, secondaryTSIGName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, secondaryZoneName)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":      secondaryZoneName,
				"master":    secondaryOtherMaster,
				"tsig_name": secondaryOtherTSIG,
				"tsig_key":  secondaryTSIGKey,
				"zone_id":   fmt.Sprintf("%d", secondaryZoneID),
			})

			updatedAt, err := time.Parse(time.RFC3339, state.Attributes["updated_at"])
			require.NoError(t, err)
			require.Equal(t, int64(secondaryUpdatedAtNS)/int64(time.Second), updatedAt.Unix())
		},
	}
}

func secondaryReadRenamedCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(secondaryAPIZoneNamed(secondaryRenamedName, nil, 0), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read overwrites id and name with the API name",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryConfig(secondaryMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, secondaryRenamedName)
			support.RequireStateAttrs(t, state, map[string]string{"name": secondaryRenamedName})
		},
	}
}

func secondaryReadTrimsIDCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(secondaryAPIZone(nil, 0), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read trims the id before the lookup",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryPaddedID,
		CurrentState: secondaryConfig(secondaryMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, secondaryZoneName)
		},
	}
}

// An empty Master fails the resource guard, so tsig_name is left alone as well.
func secondaryReadTSIGWithoutMasterCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(secondaryAPIZone(
			&dnssdk.TsigOptions{Key: secondaryOtherTSIGKey, Name: secondaryOtherTSIG},
			secondaryUpdatedAtNS,
		), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read with a tsig without a master keeps master and tsig_name",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryTSIGConfig(secondaryMaster, secondaryTSIGKey, secondaryTSIGName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"master":     secondaryMaster,
				"tsig_name":  secondaryTSIGName,
				"zone_id":    fmt.Sprintf("%d", secondaryZoneID),
				"updated_at": secondaryUpdatedAt(secondaryUpdatedAtNS),
			})
		},
	}
}

func secondaryReadWithoutTSIGCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(secondaryAPIZone(nil, 0), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read with a nil tsig refreshes zone_id and clears updated_at",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryStaleComputedState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"master":     secondaryMaster,
				"tsig_name":  secondaryTSIGName,
				"zone_id":    fmt.Sprintf("%d", secondaryZoneID),
				"updated_at": "",
			})
		},
	}
}

func secondaryReadNotFoundCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(dnssdk.SecondaryZone{}, fmt.Errorf("get secondary zone %s: %w",
			secondaryZoneName, secondaryAPIError(http.StatusNotFound, "zone not found")))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read drops a wrapped 404 zone from state",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryConfig(secondaryMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state, "state must be nil after the id is cleared")
		},
	}
}

func secondaryReadAPIFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(dnssdk.SecondaryZone{}, secondaryAPIError(http.StatusForbidden, "forbidden"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "API error on read",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryConfig(secondaryMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "get secondary zone")
			support.RequireErrorDiagContains(t, diags, "forbidden")
			support.RequireStateID(t, state, secondaryZoneName)
		},
	}
}

func secondaryReadEmptyNameCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read without an id and without a name",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentState: map[string]interface{}{},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "empty secondary zone name")
		},
	}
}

func secondaryUpdateCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	const newTSIGName = "rotated-key.secondary.example.com"

	mc.Client.On("UpdateSecondaryZone", mock.Anything, secondaryZoneName,
		mock.MatchedBy(func(req dnssdk.UpdateSecondaryZoneRequest) bool {
			return req.Master == secondaryOtherMaster &&
				req.Key == secondaryOtherTSIGKey &&
				req.Name == newTSIGName
		}),
	).Return(secondaryAPIZone(
		&dnssdk.TsigOptions{Master: secondaryOtherMaster, Name: newTSIGName},
		secondaryUpdatedAtNS,
	), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "update sends tsig_name as the request Name",
		Op:           support.OpApply,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryTSIGConfig(secondaryMaster, secondaryTSIGKey, secondaryTSIGName),
		NewConfig:    secondaryTSIGConfig(secondaryOtherMaster, secondaryOtherTSIGKey, newTSIGName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, secondaryZoneName)
			support.RequireStateAttrs(t, state, map[string]string{
				"master":     secondaryOtherMaster,
				"tsig_key":   secondaryOtherTSIGKey,
				"tsig_name":  newTSIGName,
				"zone_id":    fmt.Sprintf("%d", secondaryZoneID),
				"updated_at": secondaryUpdatedAt(secondaryUpdatedAtNS),
			})
		},
	}
}

func secondaryUpdateOverwritesMasterCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("UpdateSecondaryZone", mock.Anything, secondaryZoneName,
		mock.MatchedBy(func(req dnssdk.UpdateSecondaryZoneRequest) bool {
			return req.Master == secondaryOtherMaster
		}),
	).Return(secondaryAPIZone(
		&dnssdk.TsigOptions{Master: "203.0.113.77", Name: secondaryTSIGName},
		secondaryUpdatedAtNS,
	), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "update stores the master returned by the API, not the configured one",
		Op:           support.OpApply,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryTSIGConfig(secondaryMaster, secondaryTSIGKey, secondaryTSIGName),
		NewConfig:    secondaryTSIGConfig(secondaryOtherMaster, secondaryTSIGKey, secondaryTSIGName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"master": "203.0.113.77",
			})
		},
	}
}

func secondaryUpdateWithoutTSIGCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("UpdateSecondaryZone", mock.Anything, secondaryZoneName,
		mock.MatchedBy(func(req dnssdk.UpdateSecondaryZoneRequest) bool {
			return req.Master == secondaryOtherMaster && req.Key == "" && req.Name == ""
		}),
	).Return(secondaryAPIZone(nil, 0), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "dropping tsig from the config sends an empty key and name",
		Op:           support.OpApply,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryTSIGConfig(secondaryMaster, secondaryTSIGKey, secondaryTSIGName),
		NewConfig:    secondaryConfig(secondaryOtherMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"master":     secondaryOtherMaster,
				"tsig_name":  "",
				"updated_at": "",
			})
		},
	}
}

// Unlike create, update sends the master as configured, so spaces reach the API.
func secondaryUpdateKeepsPaddedMasterCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("UpdateSecondaryZone", mock.Anything, secondaryZoneName,
		mock.MatchedBy(func(req dnssdk.UpdateSecondaryZoneRequest) bool {
			return req.Master == secondaryPaddedMaster
		}),
	).Return(secondaryAPIZone(nil, 0), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "update does not trim the master",
		Op:           support.OpApply,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryConfig(secondaryOtherMaster),
		NewConfig:    secondaryConfig(secondaryPaddedMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"master": secondaryPaddedMaster,
			})
		},
	}
}

func secondaryUpdateAPIFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("UpdateSecondaryZone", mock.Anything, secondaryZoneName, mock.Anything).
		Return(dnssdk.SecondaryZone{}, fmt.Errorf("api error: master is unreachable"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "API error on update",
		Op:           support.OpApply,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryConfig(secondaryMaster),
		NewConfig:    secondaryConfig(secondaryOtherMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "update secondary zone")
			support.RequireErrorDiagContains(t, diags, "master is unreachable")
			support.RequireStateID(t, state, secondaryZoneName)
		},
	}
}

func secondaryDeleteCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("DeleteSecondaryZone", mock.Anything, secondaryZoneName).Return(nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "delete secondary zone",
		Op:           support.OpDelete,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryConfig(secondaryMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func secondaryDeleteNotFoundCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("DeleteSecondaryZone", mock.Anything, secondaryZoneName).
		Return(secondaryAPIError(http.StatusNotFound, "zone not found"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "delete swallows a 404",
		Op:           support.OpDelete,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryConfig(secondaryMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func secondaryDeleteAPIFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("DeleteSecondaryZone", mock.Anything, secondaryZoneName).
		Return(fmt.Errorf("api error: zone is locked"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "API error on delete keeps state",
		Op:           support.OpDelete,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    secondaryZoneName,
		CurrentState: secondaryConfig(secondaryMaster),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "delete secondary zone")
			support.RequireErrorDiagContains(t, diags, "zone is locked")
			support.RequireStateID(t, state, secondaryZoneName)
		},
	}
}

func secondaryDeleteEmptyNameCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "delete without an id and without a name",
		Op:           support.OpDelete,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentState: map[string]interface{}{},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "empty secondary zone name")
		},
	}
}

func TestIntegrationSecondaryZone_TableDriven(t *testing.T) {
	t.Parallel()

	resource := dnsResource(t, "edgecenter_dns_secondary_zone")

	cases := []support.ResourceCase[*dnsmock.MockedDNS]{
		secondaryCreateCase(),
		secondaryCreateTrimsNameAndMasterCase(),
		secondaryCreateWithoutTSIGCase(),
		secondaryCreateAdoptsExistingCase(),
		secondaryCreateLookupFailureCase(),
		secondaryCreatePointerNotFoundCase(),
		secondaryCreateAPIFailureCase(),
		secondaryReadCase(),
		secondaryReadRenamedCase(),
		secondaryReadTrimsIDCase(),
		secondaryReadTSIGWithoutMasterCase(),
		secondaryReadWithoutTSIGCase(),
		secondaryReadNotFoundCase(),
		secondaryReadAPIFailureCase(),
		secondaryReadEmptyNameCase(),
		secondaryUpdateCase(),
		secondaryUpdateOverwritesMasterCase(),
		secondaryUpdateKeepsPaddedMasterCase(),
		secondaryUpdateWithoutTSIGCase(),
		secondaryUpdateAPIFailureCase(),
		secondaryDeleteCase(),
		secondaryDeleteNotFoundCase(),
		secondaryDeleteAPIFailureCase(),
		secondaryDeleteEmptyNameCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*dnsmock.MockedDNS])
}

// The case runner cannot build a state with an empty id (ResourceData.State()
// returns nil then), so the name fallback is driven off a raw ResourceData.
func TestIntegrationSecondaryZoneReadFallsBackToName(t *testing.T) {
	t.Parallel()

	res := dnsResource(t, "edgecenter_dns_secondary_zone")

	mc := dnsmock.NewMockedDNS()
	t.Cleanup(func() { mc.MockCleanup(t) })

	mc.Client.On("GetSecondaryZone", mock.Anything, secondaryZoneName).
		Return(secondaryAPIZone(&dnssdk.TsigOptions{Master: secondaryOtherMaster}, 0), nil)

	data := schema.TestResourceDataRaw(t, res.Schema, secondaryConfig(secondaryMaster))
	require.Empty(t, data.Id())

	diags := res.ReadContext(context.Background(), data, mc.Config)

	support.RequireNoDiags(t, diags)
	require.Equal(t, secondaryZoneName, data.Id())
	require.Equal(t, secondaryOtherMaster, data.Get("master"))
}
