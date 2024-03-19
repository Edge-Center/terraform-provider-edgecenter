package edgecenter

import (
	"context"
	"strings"
	"time"

	"github.com/connerdouglass/go-retry"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const ReservedFixedIPVIPDisabledPortSecurityErrMsg = "Port Security must be enabled in order to have allowed address pairs on a port"

func assignAllowedAddressPairs(ctx context.Context, client *edgecloudV2.Client, portID string, allowedAddressPairs []interface{}) error {
	allowedAddressPairsRequest := edgecloudV2.PortsAllowedAddressPairsRequest{}
	for _, p := range allowedAddressPairs {
		pair := p.(map[string]interface{})
		allowedAddressPair := edgecloudV2.PortsAllowedAddressPairs{
			IPAddress:  pair["ip_address"].(string),
			MacAddress: pair["mac_address"].(string),
		}
		allowedAddressPairsRequest.AllowedAddressPairs = append(allowedAddressPairsRequest.AllowedAddressPairs, allowedAddressPair)
	}
	_, _, err := client.Ports.Assign(ctx, portID, &allowedAddressPairsRequest)
	if err != nil {
		return err
	}

	return nil
}

func retryReplaceInstancePorts(ctx context.Context, client *edgecloudV2.Client, portID string, addInstancePortsRequest edgecloudV2.AddInstancePortsRequest) error {
	return retry.Run(
		ctx,
		retry.Limit(4),                 // <-- Limit retries
		retry.Exponential(time.Second), // <-- Exponential backoff
		func(ctx context.Context) error {
			if _, _, err := client.ReservedFixedIP.ReplaceInstancePorts(ctx, portID, &addInstancePortsRequest); err != nil {
				if strings.Contains(err.Error(), ReservedFixedIPVIPDisabledPortSecurityErrMsg) {
					return err
				}
				return retry.RetryErr(err)
			}
			return nil
		})
}

func retryAddInstancePorts(ctx context.Context, client *edgecloudV2.Client, portID string, addInstancePortsRequest edgecloudV2.AddInstancePortsRequest) error {
	return retry.Run(
		ctx,
		retry.Limit(4),                 // <-- Limit retries
		retry.Exponential(time.Second), // <-- Exponential backoff
		func(ctx context.Context) error {
			if _, _, err := client.ReservedFixedIP.AddInstancePorts(ctx, portID, &addInstancePortsRequest); err != nil {
				if strings.Contains(err.Error(), ReservedFixedIPVIPDisabledPortSecurityErrMsg) {
					return err
				}
				return retry.RetryErr(err)
			}
			return nil
		})
}

func retryAllowedAddressPairs(ctx context.Context, client *edgecloudV2.Client, portID string, allowedAddressPairs []interface{}) error {
	return retry.Run(
		ctx,
		retry.Limit(4),                 // <-- Limit retries
		retry.Exponential(time.Second), // <-- Exponential backoff
		func(ctx context.Context) error {
			if err := assignAllowedAddressPairs(ctx, client, portID, allowedAddressPairs); err != nil {
				return retry.RetryErr(err)
			}
			return nil
		})
}
