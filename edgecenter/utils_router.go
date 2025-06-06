package edgecenter

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/router/v1/routers"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

var routerDecoderConfig = &mapstructure.DecoderConfig{
	TagName: "json",
}

// StringToNetHookFuncV2 returns a DecodeHookFunc for the mapstructure package to handle the custom
// conversion of string values to net.IP and edgecloudV2.CIDR types.
func StringToNetHookFuncV2() mapstructure.DecodeHookFuncType {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		// Only process strings as source type.
		if f.Kind() != reflect.String {
			return data, nil
		}

		// Process the target types.
		switch t {
		case reflect.TypeOf(edgecloudV2.CIDR{}):
			var ecCIDR edgecloudV2.CIDR
			_, ipNet, err := net.ParseCIDR(data.(string))
			if err != nil {
				return nil, err
			}
			ecCIDR.IP = ipNet.IP
			ecCIDR.Mask = ipNet.Mask
			return ecCIDR, nil
		case reflect.TypeOf(net.IP{}):
			ip := net.ParseIP(data.(string))
			if ip == nil {
				return nil, fmt.Errorf("failed parsing ip %v", data)
			}
			return ip, nil
		default:
			// If the target type is not supported, return the data as is.
			return data, nil
		}
	}
}

// extractHostRoutesMapV2 converts a slice of interface{} representing host routes into a slice of edgecloudV2.HostRoute.
func extractHostRoutesMapV2(v []interface{}) ([]edgecloudV2.HostRoute, error) {
	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook: StringToNetHookFuncV2(),
	}

	hostRoutes := make([]edgecloudV2.HostRoute, len(v))
	for i, hostRoute := range v {
		hs, ok := hostRoute.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("failed to assert host route as map[string]interface{}")
		}
		var H edgecloudV2.HostRoute
		err := MapStructureDecoder(&H, &hs, decoderConfig)
		if err != nil {
			return nil, err
		}
		hostRoutes[i] = H
	}

	return hostRoutes, nil
}

// routerInterfaceUniqueID generates a unique ID for a router interface using its subnet ID.
func routerInterfaceUniqueID(i interface{}) int {
	e := i.(map[string]interface{})

	subnetID := e["subnet_id"].(string)

	h := md5.New()
	io.WriteString(h, subnetID)

	return int(binary.BigEndian.Uint64(h.Sum(nil)))
}

// extractExternalGatewayInfoMap converts the first element of a gateway slice
// into a routers.GatewayInfo struct using the provided mapstructure decoder configuration.
func extractExternalGatewayInfoMap(gw []interface{}) (routers.GatewayInfo, error) {
	gateway, ok := gw[0].(map[string]interface{})
	if !ok {
		return routers.GatewayInfo{}, fmt.Errorf("failed to assert gateway as map[string]interface{}")
	}

	var gwInfo routers.GatewayInfo
	err := MapStructureDecoder(&gwInfo, &gateway, routerDecoderConfig)
	if err != nil {
		return routers.GatewayInfo{}, err
	}

	return gwInfo, nil
}

// extractExternalGatewayInfoMapV2 converts the first element of a gateway slice
// into a edgecloudV2.ExternalGatewayInfoCreate struct using the provided mapstructure decoder configuration.
func extractExternalGatewayInfoMapV2(gw []interface{}) (edgecloudV2.ExternalGatewayInfoCreate, error) {
	gateway, ok := gw[0].(map[string]interface{})
	if !ok {
		return edgecloudV2.ExternalGatewayInfoCreate{}, fmt.Errorf("failed to assert gateway as map[string]interface{}")
	}

	var gwInfo edgecloudV2.ExternalGatewayInfoCreate
	err := MapStructureDecoder(&gwInfo, &gateway, routerDecoderConfig)
	if err != nil {
		return edgecloudV2.ExternalGatewayInfoCreate{}, err
	}

	return gwInfo, nil
}

// extractInterfacesMapV2 converts a slice of interface{} representing router interfaces
// into a slice of edgecloudV2.RouterInterfaceCreate using the provided mapstructure decoder configuration.
func extractInterfacesMapV2(interfaces []interface{}) ([]edgecloudV2.RouterInterfaceCreate, error) {
	ifaceList := make([]edgecloudV2.RouterInterfaceCreate, len(interfaces))
	for i, iface := range interfaces {
		inter, ok := iface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("failed to assert interface as map[string]interface{}")
		}

		var ifaceInfo edgecloudV2.RouterInterfaceCreate
		err := MapStructureDecoder(&ifaceInfo, &inter, routerDecoderConfig)
		if err != nil {
			return nil, err
		}

		ifaceList[i] = ifaceInfo
	}

	return ifaceList, nil
}

// getRouter retrieves a router from the edge cloud service.
// It attempts to find the router either by its ID or by its name.
func getRouter(ctx context.Context, clientV2 *edgecloudV2.Client, d *schema.ResourceData) (*edgecloudV2.Router, error) {
	var (
		router *edgecloudV2.Router
		err    error
	)

	name := d.Get(NameField).(string)
	routerID := d.Get(IDField).(string)

	switch {
	case routerID != "":
		router, _, err = clientV2.Routers.Get(ctx, routerID)
		if err != nil {
			return nil, err
		}
	default:
		rs, _, err := clientV2.Routers.List(ctx)
		if err != nil {
			return nil, err
		}

		foundRouters := make([]edgecloudV2.Router, 0, len(rs))
		for _, r := range rs {
			if r.Name == name {
				foundRouters = append(foundRouters, r)
			}
		}

		switch {
		case len(foundRouters) == 0:
			return nil, errors.New("router does not exist")

		case len(foundRouters) > 1:
			var message bytes.Buffer
			message.WriteString("Found routers:\n")

			for _, rr := range foundRouters {
				message.WriteString(fmt.Sprintf("  - ID: %s\n", rr.ID))
			}

			return nil, fmt.Errorf("multiple routers found.\n %s.\n Use router ID instead of name", message.String())
		}

		router = &foundRouters[0]
	}

	return router, nil
}
