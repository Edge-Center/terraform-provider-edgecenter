package edgecenter

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	dnssdk "github.com/Edge-Center/edgecenter-dns-sdk-go"
	storageSDK "github.com/Edge-Center/edgecenter-storage-sdk-go"
	cdn "github.com/Edge-Center/edgecentercdn-go"
	eccdnProvider "github.com/Edge-Center/edgecentercdn-go/edgecenter/provider"
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	ec "github.com/Edge-Center/edgecentercloud-go/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type VarName string

const (
	EC_USERNAME_VAR           VarName = "EC_USERNAME"
	EC_PASSWORD_VAR           VarName = "EC_PASSWORD"
	EC_CDN_URL_VAR            VarName = "EC_CDN_URL"
	EC_STORAGE_URL_VAR        VarName = "EC_STORAGE_API"
	EC_DNS_URL_VAR            VarName = "EC_DNS_API"
	EC_IMAGE_VAR              VarName = "EC_IMAGE"
	EC_SECGROUP_VAR           VarName = "EC_SECGROUP"
	EC_EXT_NET_VAR            VarName = "EC_EXT_NET"
	EC_PRIV_NET_VAR           VarName = "EC_PRIV_NET"
	EC_PRIV_SUBNET_VAR        VarName = "EC_PRIV_SUBNET"
	EC_LB_ID_VAR              VarName = "EC_LB_ID"
	EC_LBLISTENER_ID_VAR      VarName = "EC_LBLISTENER_ID"
	EC_LBPOOL_ID_VAR          VarName = "EC_LBPOOL_ID"
	EC_VOLUME_ID_VAR          VarName = "EC_VOLUME_ID"
	EC_CDN_ORIGINGROUP_ID_VAR VarName = "EC_CDN_ORIGINGROUP_ID"
	EC_CDN_RESOURCE_ID_VAR    VarName = "EC_CDN_RESOURCE_ID"
	EC_NETWORK_ID_VAR         VarName = "EC_NETWORK_ID"
	EC_SUBNET_ID_VAR          VarName = "EC_SUBNET_ID"
	EC_CLUSTER_ID_VAR         VarName = "EC_CLUSTER_ID"
	EC_CLUSTER_POOL_ID_VAR    VarName = "EC_CLUSTER_POOL_ID"
)

func getEnv(name VarName) string {
	return os.Getenv(string(name))
}

var (
	EC_USERNAME           = getEnv(EC_USERNAME_VAR)
	EC_PASSWORD           = getEnv(EC_PASSWORD_VAR)
	EC_CDN_URL            = getEnv(EC_CDN_URL_VAR)
	EC_IMAGE              = getEnv(EC_IMAGE_VAR)
	EC_SECGROUP           = getEnv(EC_SECGROUP_VAR)
	EC_EXT_NET            = getEnv(EC_EXT_NET_VAR)
	EC_PRIV_NET           = getEnv(EC_PRIV_NET_VAR)
	EC_PRIV_SUBNET        = getEnv(EC_PRIV_SUBNET_VAR)
	EC_LB_ID              = getEnv(EC_LB_ID_VAR)
	EC_LBLISTENER_ID      = getEnv(EC_LBLISTENER_ID_VAR)
	EC_LBPOOL_ID          = getEnv(EC_LBPOOL_ID_VAR)
	EC_VOLUME_ID          = getEnv(EC_VOLUME_ID_VAR)
	EC_CDN_ORIGINGROUP_ID = getEnv(EC_CDN_ORIGINGROUP_ID_VAR)
	EC_CDN_RESOURCE_ID    = getEnv(EC_CDN_RESOURCE_ID_VAR)
	EC_STORAGE_API        = getEnv(EC_STORAGE_URL_VAR)
	EC_DNS_API            = getEnv(EC_DNS_URL_VAR)
	EC_NETWORK_ID         = getEnv(EC_NETWORK_ID_VAR)
	EC_SUBNET_ID          = getEnv(EC_SUBNET_ID_VAR)
	EC_CLUSTER_ID         = getEnv(EC_CLUSTER_ID_VAR)
	EC_CLUSTER_POOL_ID    = getEnv(EC_CLUSTER_POOL_ID_VAR)
)

var varsMap = map[VarName]string{
	EC_USERNAME_VAR:           EC_USERNAME,
	EC_PASSWORD_VAR:           EC_PASSWORD,
	EC_CDN_URL_VAR:            EC_CDN_URL,
	EC_IMAGE_VAR:              EC_IMAGE,
	EC_SECGROUP_VAR:           EC_SECGROUP,
	EC_EXT_NET_VAR:            EC_EXT_NET,
	EC_PRIV_NET_VAR:           EC_PRIV_NET,
	EC_PRIV_SUBNET_VAR:        EC_PRIV_SUBNET,
	EC_LB_ID_VAR:              EC_LB_ID,
	EC_LBLISTENER_ID_VAR:      EC_LBLISTENER_ID,
	EC_LBPOOL_ID_VAR:          EC_LBPOOL_ID,
	EC_VOLUME_ID_VAR:          EC_VOLUME_ID,
	EC_CDN_ORIGINGROUP_ID_VAR: EC_CDN_ORIGINGROUP_ID,
	EC_CDN_RESOURCE_ID_VAR:    EC_CDN_RESOURCE_ID,
	EC_STORAGE_URL_VAR:        EC_STORAGE_API,
	EC_DNS_URL_VAR:            EC_DNS_API,
	EC_NETWORK_ID_VAR:         EC_NETWORK_ID,
	EC_SUBNET_ID_VAR:          EC_SUBNET_ID,
	EC_CLUSTER_ID_VAR:         EC_CLUSTER_ID,
	EC_CLUSTER_POOL_ID_VAR:    EC_CLUSTER_POOL_ID,
}

func testAccPreCheckVars(t *testing.T, vars ...VarName) {
	for _, name := range vars {
		if val := varsMap[name]; val == "" {
			t.Fatalf("'%s' must be set for acceptance test", name)
		}
	}
}

var testAccProvider *schema.Provider
var testAccProviders map[string]func() (*schema.Provider, error)

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]func() (*schema.Provider, error){
		"edgecenter": func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// nolint: deadcode,unused
func testAccPreCheck(t *testing.T) {
	vars := map[string]interface{}{
		"EC_USERNAME": EC_USERNAME,
		"EC_PASSWORD": EC_PASSWORD,
	}
	for k, v := range vars {
		if v == "" {
			t.Fatalf("'%s' must be set for acceptance test", k)
		}
	}
	checkNameAndID("PROJECT", t)
	checkNameAndID("REGION", t)
}

// nolint: unused
func checkNameAndID(resourceType string, t *testing.T) {
	// resourceType is a word in capital letters
	keyID := fmt.Sprintf("TEST_%s_ID", resourceType)
	keyName := fmt.Sprintf("TEST_%s_NAME", resourceType)
	_, haveID := os.LookupEnv(keyID)
	_, haveName := os.LookupEnv(keyName)
	if !haveID && !haveName {
		t.Fatalf("%s or %s must be set for acceptance tests", keyID, keyName)
	}
	if haveID && haveName {
		t.Fatalf("Use only one from environment variables: %s or %s", keyID, keyName)
	}
}

// nolint: deadcode,unused
func regionInfo() string {
	return objectInfo("REGION")
}

// nolint: deadcode,unused
func projectInfo() string {
	return objectInfo("PROJECT")
}

// nolint: unused
func objectInfo(resourceType string) string {
	// resourceType is a word in capital letters
	keyID := fmt.Sprintf("TEST_%s_ID", resourceType)
	keyName := fmt.Sprintf("TEST_%s_NAME", resourceType)
	if objectID, exists := os.LookupEnv(keyID); exists {
		return fmt.Sprintf(`%s_id = %s`, strings.ToLower(resourceType), objectID)
	}
	return fmt.Sprintf(`%s_name = "%s"`, strings.ToLower(resourceType), os.Getenv(keyName))
}

func CreateTestClient(provider *edgecloud.ProviderClient, endpoint, version string) (*edgecloud.ServiceClient, error) {
	projectID := 0
	var err error
	if strProjectID, exists := os.LookupEnv("TEST_PROJECT_ID"); exists {
		projectID, err = strconv.Atoi(strProjectID)
		if err != nil {
			return nil, err
		}
	} else {
		projectID, err = GetProject(provider, 0, os.Getenv("TEST_PROJECT_NAME"))
		if err != nil {
			return nil, err
		}
	}
	regionID := 0
	if strRegionID, exists := os.LookupEnv("TEST_REGION_ID"); exists {
		regionID, err = strconv.Atoi(strRegionID)
		if err != nil {
			return nil, err
		}
	} else {
		regionID, err = GetProject(provider, 0, os.Getenv("TEST_REGION_NAME"))
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

func createTestConfig() (*Config, error) {
	provider, err := ec.AuthenticatedClient(edgecloud.AuthOptions{
		APIURL:      os.Getenv("EC_API"),
		AuthURL:     os.Getenv("EC_PLATFORM"),
		Username:    os.Getenv("EC_USERNAME"),
		Password:    os.Getenv("EC_PASSWORD"),
		AllowReauth: true,
	})
	if err != nil {
		return nil, err
	}

	cdnProvider := eccdnProvider.NewClient(EC_CDN_URL, eccdnProvider.WithSignerFunc(func(req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+provider.AccessToken())
		return nil
	}))
	cdnService := cdn.NewService(cdnProvider)

	storageAPI := EC_STORAGE_API
	stHost, stPath, err := ExtractHostAndPath(storageAPI)
	var storageClient *storageSDK.SDK
	if err == nil {
		storageClient = storageSDK.NewSDK(stHost, stPath, storageSDK.WithBearerAuth(provider.AccessToken))
	}

	var dnsClient *dnssdk.Client
	if EC_DNS_API != "" {
		baseUrl, err := url.Parse(EC_DNS_API)
		if err == nil {
			authorizer := dnssdk.BearerAuth(provider.AccessToken())
			dnsClient = dnssdk.NewClient(authorizer, func(client *dnssdk.Client) {
				client.BaseURL = baseUrl
			})
		}

	}

	config := Config{
		Provider:      provider,
		CDNClient:     cdnService,
		StorageClient: storageClient,
		DNSClient:     dnsClient,
	}

	return &config, nil
}

func testAccCheckResourceExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// retrieve the resource by name from state
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Widget ID is not set")
		}
		return nil
	}
}
