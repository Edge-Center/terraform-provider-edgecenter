package edgecenter

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	dnssdk "github.com/Edge-Center/edgecenter-dns-sdk-go"
	storageSDK "github.com/Edge-Center/edgecenter-storage-sdk-go"
	cdn "github.com/Edge-Center/edgecentercdn-go"
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	ec "github.com/Edge-Center/edgecentercloud-go/edgecenter"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/instances"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/types"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/lbpools"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/listeners"
	typesLb "github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/types"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/availablenetworks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/project/v1/projects"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/region/v1/regions"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/router/v1/routers"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/securitygroup/v1/securitygroups"
	typesSG "github.com/Edge-Center/edgecentercloud-go/edgecenter/securitygroup/v1/types"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
)

const (
	versionPointV1 = "v1"
	versionPointV2 = "v2"

	projectPoint = "projects"
	regionPoint  = "regions"
)

type Config struct {
	Provider      *edgecloud.ProviderClient
	CDNClient     cdn.ClientService
	StorageClient *storageSDK.SDK
	DNSClient     *dnssdk.Client
}

type Project struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Projects struct {
	Count   int       `json:"count"`
	Results []Project `json:"results"`
}

type Region struct {
	Id          int    `json:"id"`
	DisplayName string `json:"display_name"`
}

type Regions struct {
	Count   int      `json:"count"`
	Results []Region `json:"results"`
}

var config = &mapstructure.DecoderConfig{
	TagName: "json",
}

type instanceInterfaces []interface{}

func (s instanceInterfaces) Len() int {
	return len(s)
}

func (s instanceInterfaces) Less(i, j int) bool {
	ifLeft := s[i].(map[string]interface{})
	ifRight := s[j].(map[string]interface{})

	// only bm instance has a parent interface, and it should be attached first
	isTrunkLeft, okLeft := ifLeft["is_parent"]
	isTrunkRight, okRight := ifRight["is_parent"]
	if okLeft && okRight {
		left, _ := isTrunkLeft.(bool)
		right, _ := isTrunkRight.(bool)
		switch {
		case left && !right:
			return true
		case right && !left:
			return false
		}
	}

	lOrder, _ := ifLeft["order"].(int)
	rOrder, _ := ifRight["order"].(int)
	return lOrder < rOrder
}

func (s instanceInterfaces) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func MapStructureDecoder(strct interface{}, v *map[string]interface{}, config *mapstructure.DecoderConfig) error {
	config.Result = strct
	decoder, _ := mapstructure.NewDecoder(config)
	err := decoder.Decode(*v)
	if err != nil {
		return err
	}
	return nil
}

func StringToNetHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t == reflect.TypeOf(edgecloud.CIDR{}) {
			var eccidr edgecloud.CIDR
			_, ipNet, err := net.ParseCIDR(data.(string))
			eccidr.IP = ipNet.IP
			eccidr.Mask = ipNet.Mask
			return eccidr, err
		}
		if t == reflect.TypeOf(net.IP{}) {
			ip := net.ParseIP(data.(string))
			if ip == nil {
				return net.IP{}, fmt.Errorf("failed parsing ip %v", data)
			}
			return ip, nil
		}
		return data, nil
	}
}

func extractHostRoutesMap(v []interface{}) ([]subnets.HostRoute, error) {
	var config = &mapstructure.DecoderConfig{
		DecodeHook: StringToNetHookFunc(),
	}

	HostRoutes := make([]subnets.HostRoute, len(v))
	for i, hostroute := range v {
		hs := hostroute.(map[string]interface{})
		var H subnets.HostRoute
		err := MapStructureDecoder(&H, &hs, config)
		if err != nil {
			return nil, err
		}
		HostRoutes[i] = H
	}
	return HostRoutes, nil
}

func extractExternalGatewayInfoMap(gw []interface{}) (routers.GatewayInfo, error) {
	gateway := gw[0].(map[string]interface{})
	var GW routers.GatewayInfo
	err := MapStructureDecoder(&GW, &gateway, config)
	if err != nil {
		return GW, err
	}
	return GW, nil
}

func extractInterfacesMap(interfaces []interface{}) ([]routers.Interface, error) {
	Interfaces := make([]routers.Interface, len(interfaces))
	for i, iface := range interfaces {
		inter := iface.(map[string]interface{})
		var I routers.Interface
		err := MapStructureDecoder(&I, &inter, config)
		if err != nil {
			return nil, err
		}
		Interfaces[i] = I
	}
	return Interfaces, nil
}

func extractVolumesMap(volumes []interface{}) ([]instances.CreateVolumeOpts, error) {
	Volumes := make([]instances.CreateVolumeOpts, len(volumes))
	for i, volume := range volumes {
		vol := volume.(map[string]interface{})
		var V instances.CreateVolumeOpts
		err := MapStructureDecoder(&V, &vol, config)
		if err != nil {
			return nil, err
		}
		Volumes[i] = V
	}
	return Volumes, nil
}

// todo refactoring
func extractVolumesIntoMap(volumes []interface{}) map[string]map[string]interface{} {
	Volumes := make(map[string]map[string]interface{}, len(volumes))
	for _, volume := range volumes {
		vol := volume.(map[string]interface{})
		Volumes[vol["volume_id"].(string)] = vol
	}
	return Volumes
}

func extractInstanceVolumesMap(volumes []interface{}) map[string]bool {
	result := make(map[string]bool)
	for _, volume := range volumes {
		v := volume.(map[string]interface{})
		result[v["volume_id"].(string)] = true
	}
	return result
}

func extractInstanceInterfacesMap(interfaces []interface{}) ([]instances.InterfaceInstanceCreateOpts, error) {
	Interfaces := make([]instances.InterfaceInstanceCreateOpts, len(interfaces))
	for i, iface := range interfaces {
		inter := iface.(map[string]interface{})

		var I instances.InterfaceOpts
		err := MapStructureDecoder(&I, &inter, config)
		if err != nil {
			return nil, err
		}

		if inter["fip_source"] != "" {
			var fip instances.CreateNewInterfaceFloatingIPOpts
			if inter["existing_fip_id"] != "" {
				fip.Source = types.ExistingFloatingIP
				fip.ExistingFloatingID = inter["existing_fip_id"].(string)
			} else {
				fip.Source = types.NewFloatingIP
			}
			I.FloatingIP = &fip
		}

		rawSgsID := inter["security_groups"].([]interface{})
		sgs := make([]edgecloud.ItemID, len(rawSgsID))
		for i, sgID := range rawSgsID {
			sgs[i] = edgecloud.ItemID{ID: sgID.(string)}
		}

		Interfaces[i] = instances.InterfaceInstanceCreateOpts{
			InterfaceOpts:  I,
			SecurityGroups: sgs,
		}
	}
	return Interfaces, nil
}

type OrderedInterfaceOpts struct {
	instances.InterfaceOpts
	Order int
}

// todo refactoring
func extractInstanceInterfaceIntoMap(interfaces []interface{}) (map[string]OrderedInterfaceOpts, error) {
	Interfaces := make(map[string]OrderedInterfaceOpts)
	for _, iface := range interfaces {
		if iface == nil {
			continue
		}
		inter := iface.(map[string]interface{})

		var I instances.InterfaceOpts
		err := MapStructureDecoder(&I, &inter, config)
		if err != nil {
			return nil, err
		}

		if inter["fip_source"] != "" {
			var fip instances.CreateNewInterfaceFloatingIPOpts
			if inter["existing_fip_id"] != "" {
				fip.Source = types.ExistingFloatingIP
				fip.ExistingFloatingID = inter["existing_fip_id"].(string)
			} else {
				fip.Source = types.NewFloatingIP
			}
			I.FloatingIP = &fip
		}
		o, _ := inter["order"].(int)
		orderedInt := OrderedInterfaceOpts{I, o}
		Interfaces[I.SubnetID] = orderedInt
		Interfaces[I.NetworkID] = orderedInt
		Interfaces[I.PortID] = orderedInt
		if I.Type == types.ExternalInterfaceType {
			Interfaces[I.Type.String()] = orderedInt
		}
	}
	return Interfaces, nil
}

func extractKeyValue(metadata []interface{}) (instances.MetadataSetOpts, error) {
	MetaData := make([]instances.MetadataOpts, len(metadata))
	var MetadataSetOpts instances.MetadataSetOpts
	for i, meta := range metadata {
		md := meta.(map[string]interface{})
		var MD instances.MetadataOpts
		err := MapStructureDecoder(&MD, &md, config)
		if err != nil {
			return MetadataSetOpts, err
		}
		MetaData[i] = MD
	}
	MetadataSetOpts.Metadata = MetaData
	return MetadataSetOpts, nil
}

func extractMetadataMap(metadata map[string]interface{}) instances.MetadataSetOpts {
	result := make([]instances.MetadataOpts, 0, len(metadata))
	var MetadataSetOpts instances.MetadataSetOpts
	for k, v := range metadata {
		result = append(result, instances.MetadataOpts{Key: k, Value: v.(string)})
	}
	MetadataSetOpts.Metadata = result
	return MetadataSetOpts
}

func findProjectByName(arr []projects.Project, name string) (int, error) {
	for _, el := range arr {
		if el.Name == name {
			return el.ID, nil
		}
	}
	return 0, fmt.Errorf("project with name %s not found", name)
}

// GetProject returns valid projectID for a resource
func GetProject(provider *edgecloud.ProviderClient, projectID int, projectName string) (int, error) {
	log.Println("[DEBUG] Try to get project ID")
	// valid cases
	if projectID != 0 {
		return projectID, nil
	}
	client, err := ec.ClientServiceFromProvider(provider, edgecloud.EndpointOpts{
		Name:    projectPoint,
		Region:  0,
		Project: 0,
		Version: "v1",
	})
	if err != nil {
		return 0, err
	}
	projects, err := projects.ListAll(client)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] Projects: %v", projects)
	projectID, err = findProjectByName(projects, projectName)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] The attempt to get the project is successful: projectID=%d", projectID)
	return projectID, nil
}

func findRegionByName(arr []regions.Region, name string) (int, error) {
	for _, el := range arr {
		if el.DisplayName == name {
			return el.ID, nil
		}
	}
	return 0, fmt.Errorf("region with name %s not found", name)
}

// GetRegion returns valid regionID for a resource
func GetRegion(provider *edgecloud.ProviderClient, regionID int, regionName string) (int, error) {
	// valid cases
	if regionID != 0 {
		return regionID, nil
	}
	client, err := ec.ClientServiceFromProvider(provider, edgecloud.EndpointOpts{
		Name:    regionPoint,
		Region:  0,
		Project: 0,
		Version: "v1",
	})
	if err != nil {
		return 0, err
	}

	rs, err := regions.ListAll(client)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] Regions: %v", rs)
	regionID, err = findRegionByName(rs, regionName)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] The attempt to get the region is successful: regionID=%d", regionID)
	return regionID, nil
}

// ImportStringParser is a helper function for the import module. It parses check and parse an input command line string (id part).
func ImportStringParser(infoStr string) (int, int, string, error) {
	log.Printf("[DEBUG] Input id string: %s", infoStr)
	infoStrings := strings.Split(infoStr, ":")
	if len(infoStrings) != 3 {
		return 0, 0, "", fmt.Errorf("Failed import: wrong input id: %s", infoStr)

	}
	projectID, err := strconv.Atoi(infoStrings[0])
	if err != nil {
		return 0, 0, "", err
	}
	regionID, err := strconv.Atoi(infoStrings[1])
	if err != nil {
		return 0, 0, "", err
	}
	return projectID, regionID, infoStrings[2], nil
}

// ImportStringParserExtended is a helper function for the import module. It parses check and parse an input command line string (id part).
// Uses for import where need four elements, e. g. k8s pool(cluster_id), lb_member(lbpool_id).
func ImportStringParserExtended(infoStr string) (int, int, string, string, error) {
	log.Printf("[DEBUG] Input id string: %s", infoStr)
	infoStrings := strings.Split(infoStr, ":")
	if len(infoStrings) != 4 {
		return 0, 0, "", "", fmt.Errorf("Failed import: wrong input id: %s", infoStr)

	}
	projectID, err := strconv.Atoi(infoStrings[0])
	if err != nil {
		return 0, 0, "", "", err
	}
	regionID, err := strconv.Atoi(infoStrings[1])
	if err != nil {
		return 0, 0, "", "", err
	}
	return projectID, regionID, infoStrings[2], infoStrings[3], nil
}

func CreateClient(provider *edgecloud.ProviderClient, d *schema.ResourceData, endpoint string, version string) (*edgecloud.ServiceClient, error) {
	projectID, err := GetProject(provider, d.Get("project_id").(int), d.Get("project_name").(string))
	if err != nil {
		return nil, err
	}

	var regionID int
	rawRegionID := d.Get("region_id")
	rawRegionName := d.Get("region_name")
	if rawRegionID != nil && rawRegionName != nil {
		regionID, err = GetRegion(provider, rawRegionID.(int), rawRegionName.(string))
		if err != nil {
			return nil, err
		}
	}

	client, err := ec.ClientServiceFromProvider(provider, edgecloud.EndpointOpts{
		Name:    endpoint,
		Region:  regionID,
		Project: projectID,
		Version: version,
	})

	if err != nil {
		return nil, err
	}
	return client, nil
}

func revertState(d *schema.ResourceData, fields *[]string) {
	if d.Get("last_updated").(string) != "" {
		for _, field := range *fields {
			if d.HasChange(field) {
				oldValue, _ := d.GetChange(field)
				switch v := oldValue.(type) {
				case int:
					d.Set(field, v)
				case string:
					d.Set(field, v)
				case map[string]interface{}:
					d.Set(field, v)
				}
			}
			log.Printf("[DEBUG] Revert (%s) '%s' field", d.Id(), field)
		}
	}
}

func extractSessionPersistenceMap(d *schema.ResourceData) *lbpools.CreateSessionPersistenceOpts {
	var sessionOpts *lbpools.CreateSessionPersistenceOpts
	sessionPers := d.Get("session_persistence").([]interface{})
	if len(sessionPers) > 0 {
		sm := sessionPers[0].(map[string]interface{})
		sessionOpts = &lbpools.CreateSessionPersistenceOpts{
			Type: typesLb.PersistenceType(sm["type"].(string)),
		}

		granularity := sm["persistence_granularity"]
		if granularity != nil {
			sessionOpts.PersistenceGranularity = granularity.(string)
		}

		timeout := sm["persistence_timeout"]
		if timeout != nil {
			sessionOpts.PersistenceTimeout = timeout.(int)
		}

		cookieName := sm["cookie_name"]
		if cookieName != nil {
			sessionOpts.CookieName = cookieName.(string)
		}
	}
	return sessionOpts
}

func extractHealthMonitorMap(d *schema.ResourceData) *lbpools.CreateHealthMonitorOpts {
	var healthOpts *lbpools.CreateHealthMonitorOpts
	monitors := d.Get("health_monitor").([]interface{})
	if len(monitors) > 0 {
		hm := monitors[0].(map[string]interface{})
		healthOpts = &lbpools.CreateHealthMonitorOpts{
			Type:       typesLb.HealthMonitorType(hm["type"].(string)),
			Delay:      hm["delay"].(int),
			MaxRetries: hm["max_retries"].(int),
			Timeout:    hm["timeout"].(int),
		}

		maxRetriesDown := hm["max_retries_down"].(int)
		if maxRetriesDown != 0 {
			healthOpts.MaxRetriesDown = maxRetriesDown
		}

		httpMethod := hm["http_method"].(string)
		if httpMethod != "" {
			healthOpts.HTTPMethod = typesLb.HTTPMethodPointer(typesLb.HTTPMethod(httpMethod))
		}

		urlPath := hm["url_path"].(string)
		if urlPath != "" {
			healthOpts.URLPath = urlPath
		}

		expectedCodes := hm["expected_codes"].(string)
		if expectedCodes != "" {
			healthOpts.ExpectedCodes = expectedCodes
		}

		id := hm["id"].(string)
		if id != "" {
			healthOpts.ID = id
		}
	}
	return healthOpts
}

func routerInterfaceUniqueID(i interface{}) int {
	e := i.(map[string]interface{})
	h := md5.New()
	io.WriteString(h, e["subnet_id"].(string))
	return int(binary.BigEndian.Uint64(h.Sum(nil)))
}

func volumeUniqueID(i interface{}) int {
	e := i.(map[string]interface{})
	h := md5.New()
	io.WriteString(h, e["volume_id"].(string))
	return int(binary.BigEndian.Uint64(h.Sum(nil)))
}

func secGroupUniqueID(i interface{}) int {
	e := i.(map[string]interface{})
	h := md5.New()
	proto, _ := e["protocol"].(string)
	io.WriteString(h, e["direction"].(string))
	io.WriteString(h, e["ethertype"].(string))
	io.WriteString(h, proto)
	io.WriteString(h, strconv.Itoa(e["port_range_min"].(int)))
	io.WriteString(h, strconv.Itoa(e["port_range_max"].(int)))
	io.WriteString(h, e["description"].(string))
	io.WriteString(h, e["remote_ip_prefix"].(string))

	return int(binary.BigEndian.Uint64(h.Sum(nil)))
}

func validatePortRange(v interface{}, path cty.Path) diag.Diagnostics {
	val := v.(int)
	if val >= minPort && val <= maxPort {
		return nil
	}
	return diag.Errorf("available range %d-%d", minPort, maxPort)
}

func extractSecurityGroupRuleMap(r interface{}, gid string) securitygroups.CreateRuleOptsBuilder {
	rule := r.(map[string]interface{})
	opts := securitygroups.CreateSecurityGroupRuleOpts{
		Direction:       typesSG.RuleDirection(rule["direction"].(string)),
		EtherType:       typesSG.EtherType(rule["ethertype"].(string)),
		Protocol:        typesSG.Protocol(rule["protocol"].(string)),
		SecurityGroupID: &gid,
	}
	minP, maxP := rule["port_range_min"].(int), rule["port_range_max"].(int)
	if minP != 0 && maxP != 0 {
		opts.PortRangeMin = &minP
		opts.PortRangeMax = &maxP
	}

	descr, _ := rule["description"].(string)
	opts.Description = &descr

	remoteIPPrefix := rule["remote_ip_prefix"].(string)
	if remoteIPPrefix != "" {
		opts.RemoteIPPrefix = &remoteIPPrefix
	}
	return opts
}

// technical debt
func findNetworkByName(name string, nets []networks.Network) (networks.Network, bool) {
	var found bool
	var network networks.Network
	for _, n := range nets {
		if n.Name == name {
			network = n
			found = true
			break
		}
	}
	return network, found
}

// technical debt
func findSharedNetworkByName(name string, nets []availablenetworks.Network) (availablenetworks.Network, bool) {
	var found bool
	var network availablenetworks.Network
	for _, n := range nets {
		if n.Name == name {
			network = n
			found = true
			break
		}
	}
	return network, found
}

func StructToMap(obj interface{}) (newMap map[string]interface{}, err error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &newMap)
	return
}

// ExtractHostAndPath from url
func ExtractHostAndPath(uri string) (host, path string, err error) {
	if uri == "" {
		return "", "", fmt.Errorf("empty uri")
	}
	strings.Split(uri, "://")
	pUrl, err := url.Parse(uri)
	if err != nil {
		return "", "", fmt.Errorf("url parse: %w", err)
	}
	return pUrl.Scheme + "://" + pUrl.Host, pUrl.Path, nil
}

func parseCIDRFromString(cidr string) (edgecloud.CIDR, error) {
	var eccidr edgecloud.CIDR
	_, netIPNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return eccidr, err
	}
	eccidr.IP = netIPNet.IP
	eccidr.Mask = netIPNet.Mask
	return eccidr, nil
}

func isInterfaceAttached(ifs []instances.Interface, ifs2 map[string]interface{}) bool {
	subnetID, _ := ifs2["subnet_id"].(string)
	iType := types.InterfaceType(ifs2["type"].(string))
	for _, i := range ifs {
		if iType == types.ExternalInterfaceType && i.NetworkDetails.External {
			return true
		}
		for _, assignement := range i.IPAssignments {
			if assignement.SubnetID == subnetID {
				return true
			}
		}
		for _, subPort := range i.SubPorts {
			if iType == types.ExternalInterfaceType && subPort.NetworkDetails.External {
				return true
			}
			for _, assignement := range subPort.IPAssignments {
				if assignement.SubnetID == subnetID {
					return true
				}
			}
		}
	}
	return false
}

func isInterfaceContains(verifiable map[string]interface{}, ifsSet []interface{}) bool {
	verifiableType := verifiable["type"].(string)
	verifiableSubnetID, _ := verifiable["subnet_id"].(string)
	for _, e := range ifsSet {
		i := e.(map[string]interface{})
		iType := i["type"].(string)
		subnetID, _ := i["subnet_id"].(string)
		if iType == types.ExternalInterfaceType.String() && verifiableType == types.ExternalInterfaceType.String() {
			return true
		}

		if iType == verifiableType {
			if subnetID == verifiableSubnetID {
				return true
			}
		}
	}
	return false
}

func extractListenerIntoMap(listener *listeners.Listener) map[string]interface{} {
	l := make(map[string]interface{})
	l["id"] = listener.ID
	l["name"] = listener.Name
	l["protocol"] = listener.Protocol.String()
	l["protocol_port"] = listener.ProtocolPort
	l["secret_id"] = listener.SecretID
	l["sni_secret_id"] = listener.SNISecretID
	return l
}

func getUniqueID(d *schema.ResourceData) string {
	return fmt.Sprintf(
		"%d%d%s%s",
		d.Get("region_id").(int),
		d.Get("project_id").(int),
		d.Get("region_name").(string),
		d.Get("project_name").(string),
	)
}