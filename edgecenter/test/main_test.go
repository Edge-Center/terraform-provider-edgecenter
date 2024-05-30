package edgecenter_test

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	dnssdk "github.com/Edge-Center/edgecenter-dns-sdk-go"
	storageSDK "github.com/Edge-Center/edgecenter-storage-sdk-go"
	cdn "github.com/Edge-Center/edgecentercdn-go"
	eccdnProvider "github.com/Edge-Center/edgecentercdn-go/edgecenter/provider"
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	ec "github.com/Edge-Center/edgecentercloud-go/edgecenter"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/volume/v1/volumes"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

//nolint:unused
const (
	instanceTestName   = "test-vm"
	instanceV2TestName = "test-vmV2"
	clusterTestName    = "test-cluster"
	poolTestName       = "test-pool"
	lbTestName         = "test-lb"
	lbListenerTestName = "test-listener"
	networkTestName    = "test-network"
	subnetTestName     = "test-subnet"
	volumeTestName     = "test-volume"
	secretTestName     = "test-secret"
	kpTestName         = "test-kp"

	flavorTest           = "g1-standard-1-2"
	osDistroTest         = "debian"
	clusterVersionTest   = "1.20.15"
	cidrTest             = "192.168.42.0/24"
	nodeCountTest        = 1
	dockerVolumeSizeTest = 10
	ockerVolumeTypeTest  = volumes.Standard
	minNodeCountTest     = 1
	maxNodeCountTest     = 1
	volumeSizeTest       = 1

	pkTest           = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC1bdbQYquD/swsZpFPXagY9KvhlNUTKYMdhRNtlGglAMgRxJS3Q0V74BNElJtP+UU/AbZD4H2ZAwW3PLLD/maclnLlrA48xg/ez9IhppBop0WADZ/nB4EcvQfR/Db7nHDTZERW6EiiGhV6CkHVasK2sY/WNRXqPveeWUlwCqtSnU90l/s9kQCoEfkM2auO6ppJkVrXbs26vcRclS8KL7Cff4HwdVpV7b+edT5seZdtrFUCbkEof9D9nGpahNvg8mYWf0ofx4ona4kaXm1NdPID+ljvE/dbYUX8WZRmyLjMvVQS+VxDJtsiDQIVtwbC4w+recqwDvHhLWwoeczsbEsp ondi@ds`
	privateKey       = "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDQ4E6U0vql4EST\n8o41TlHRz6MKmMhddVUjM2juTKjxv4WuB4T3z/wokznEjQg4H7gfYEKeCJqelrfq\ntdOtbPsznSceMOXB5uA2Sc9WVKwk7owoRJxPd4LQeOcarVOFdIzudzkgSK/oV7Za\nL8Y2hylsB4SX2cfbULtmW/WDePp3YZAL6zYV1fXJSnK+hL2iUSqikiViEGRta+47\nnaTKZnnmSgojdshzsw0wlF/PgRJ/Anf9j9J8ratdJP81yAG5daU3L2NdJ3qx9UbV\ntKnSq2z2u4yx6xdb4t4WFQBKNjC6+YZN/gI5lp96p3FNTNS4PKYxAAUrnCwf0EE3\n7dOR4eWlAgMBAAECggEBALPm3ge0h4li1e4PVYh4AmSRT74KxVgpfMCqwM+uWzyM\nVpkDhPTjwC06UOEHD3M3bqAninkOtA2vhoyzOrP+T4Wu70hDmUAemDJp9BhJKVNN\n2o28Olz/dD4WRAZoDq29Kr0hFqTFtiyJj1eyGihQ1c5j00HuowI0UJPi1Fz+T8uN\nPwukUtTPYwEds6SApii3v9VKjmvbRDmsbHU3KkUoaeqpRnRagyp1vtoLXigezUcK\nrQcoh6wlKtvj0YLR2lxq9Wmj1nn6m3F5Bom54X8o18tcOmFSRudRb+Fxjb0jnqSK\nAsyVlZg4alTBQUmx9gIKv0oSJAIh2nXdclECkGjs8WkCgYEA9xvdDWephsbv+X3k\nndnDG9JTxfrR6HMHPrUrTaZ8/VD+Qw4zuReoNGkcQbV3Cb26egprWQWfYc9+l6mU\nAWgOjFgeGie1uwOwkhv6CfhE/iVvotJ3hOOsC5pLEhz4vRpO75C9wSehjfTYkP1m\nXEAhRTRbgMnvzChWyh5CEjosX5sCgYEA2GRHrG0JVxsYSCugLPKf9fSK4CQDm0bK\nywBwZtAWX0xhiHO/BW6PeK1Mqx2nbiWl1hXNpZKJNS9bnrZWym/yUqOvg2XJKjb6\nhHBvwAD1MOQ8Ysby4JHGCrMBEwlcDpI2wpMpXkKhU3X0XWjkqrhqCH/TETFKkqLt\nfJX/c9PTQ78CgYAEPek0grQJST7zVHLpNsS/pIOloWGbEOZt8CQ3KAV7P7mtov/G\nTJ6pj6hZhGjvtN8Pm0Aufgc3YZ11swaEY6nkRNr3bfkTpcORLoPDSgy9JB1feSdu\nE45vgI2LWQ34CQyT1jM7rpd6XVqeWos4SC2KB5UOh+ji40piG9TchT0fwwKBgA/M\nmpMTTvhGKSqzzLkbaeR6W11sI7tFmu7hdFN9Y/THTeO5l7vcy6ri9FMWEjBvnUEZ\nTG+HWG9CquzWoVWcgNPZ0anFV7+2Teo3j2E0cLKGJ4aKwhb1bcFAOpbaOxdxQ4BH\nYGDaeo7ucM4VJ4TzfAJs2stJjwlPzgknpoQddjJfAoGBAIFfnU8x/SrNhAqZrG9d\n3kpJ5LmbVswOYtj01KHM+KpEwOQVF+s2NOeHqyC7QUIWrue00+1MT88F9cNHDeWk\n0dEOJNWCfzcV85l8A+0p6/4qAW7h7RNiFqeA8GyVKCT8f7fu/7WpYw8D0aq8w5X/\nKZl+AjB+MzYFs71+SC4ohTlI\n-----END PRIVATE KEY-----"
	certificate      = "-----BEGIN CERTIFICATE-----\nMIIDpDCCAoygAwIBAgIJAIUvym0uaBHbMA0GCSqGSIb3DQEBCwUAMD0xCzAJBgNV\nBAYTAlJVMQ8wDQYDVQQIDAZNT1NDT1cxCzAJBgNVBAoMAkNBMRAwDgYDVQQDDAdS\nT09UIENBMB4XDTIxMDczMDE1MTU0NVoXDTMxMDcyODE1MTU0NVowTDELMAkGA1UE\nBhMCQ0ExDTALBgNVBAgMBE5vbmUxCzAJBgNVBAcMAk5CMQ0wCwYDVQQKDAROb25l\nMRIwEAYDVQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEK\nAoIBAQDQ4E6U0vql4EST8o41TlHRz6MKmMhddVUjM2juTKjxv4WuB4T3z/wokznE\njQg4H7gfYEKeCJqelrfqtdOtbPsznSceMOXB5uA2Sc9WVKwk7owoRJxPd4LQeOca\nrVOFdIzudzkgSK/oV7ZaL8Y2hylsB4SX2cfbULtmW/WDePp3YZAL6zYV1fXJSnK+\nhL2iUSqikiViEGRta+47naTKZnnmSgojdshzsw0wlF/PgRJ/Anf9j9J8ratdJP81\nyAG5daU3L2NdJ3qx9UbVtKnSq2z2u4yx6xdb4t4WFQBKNjC6+YZN/gI5lp96p3FN\nTNS4PKYxAAUrnCwf0EE37dOR4eWlAgMBAAGjgZcwgZQwVwYDVR0jBFAwTqFBpD8w\nPTELMAkGA1UEBhMCUlUxDzANBgNVBAgMBk1PU0NPVzELMAkGA1UECgwCQ0ExEDAO\nBgNVBAMMB1JPT1QgQ0GCCQCectJTETy4lTAJBgNVHRMEAjAAMAsGA1UdDwQEAwIE\n8DAhBgNVHREEGjAYgglsb2NhbGhvc3SCCyoubG9jYWxob3N0MA0GCSqGSIb3DQEB\nCwUAA4IBAQBqzJcwygLsVCTPlReUpcKVn84aFqzfZA0m7hYvH+7PDH/FM8SbX3zg\nteBL/PgQAZw1amO8xjeMc2Pe2kvi9VrpfTeGqNia/9axhGu3q/NEP0tyDFXAE2bR\njBdGhd5gCmg+X4WdHigCgn51cz5r2k3fSOIWP+TQWHqc8Yt+vZXnkwnQkRA1Ki7N\nWOiJjj/ae5RWwma/kJNmShTZn754gbQn06bAjNbPjclsHRLkawmLqikd1rYUhIdk\nOr1Nrl+CWMx3CXg0TVVdJ6rH3dO31uyvb+3qEY7WnL+HhZyr08ay8gJsEKPuPFA2\nxvveXqt9ceU5qh+8T7mHwGALEUw96QcP\n-----END CERTIFICATE-----"
	certificateChain = "-----BEGIN CERTIFICATE-----\nMIIC9jCCAd4CCQCectJTETy4lTANBgkqhkiG9w0BAQsFADA9MQswCQYDVQQGEwJS\nVTEPMA0GA1UECAwGTU9TQ09XMQswCQYDVQQKDAJDQTEQMA4GA1UEAwwHUk9PVCBD\nQTAeFw0yMTA3MzAxNTExMzVaFw0yNDA1MTkxNTExMzVaMD0xCzAJBgNVBAYTAlJV\nMQ8wDQYDVQQIDAZNT1NDT1cxCzAJBgNVBAoMAkNBMRAwDgYDVQQDDAdST09UIENB\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAo6tZ0NV6QIR/mvsqtAII\nzTTuBMrZR5OTwKvcGnhe4GVDwzJ/OgEWkghLAzOojcJvkfzJOtWwOXqwgphksc+7\n+vwIPTPt3iWjbQUzXK8pFLkjxrO8px/QxPuUrp+U6DTVvvgQesjMZ9jQRUFKOiCc\nu0st1N5Q/CJR4VOJxtYoLy1ZUlsABhwJ+6trkoOFTLRPlMUX1EIG57jYAotHvQFo\nc8UNx3KzvJsJJ56SniXCIkeu61IOt8aOXHU+3TLYhZnPiP311cMbXA0J3vGPRZwz\n25BZjF3IF/ShXlfzz76FjWUTAThc0+HA8lzx53xD4/n8HN+sGubGx9TvLyZimG/U\nGwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQAnK8Wzw33fR6R6pqV05XI9Yu8J+BwC\nCn2bKxxYwwQWZyX1as+UIlGuvyBRJba9W2UGMj95FQfWVdDyFC98spUur+O/5yL+\nNHH+dxGnkxIRc6RMIy+GXJwPrLiB/t70hSvwgVa249zNJVcwYN/5SGX5wLaJKnim\neY99xm75nr03O/RJK/DR8HvWysH7zxvrMWs0ppfwxkxrwOcg0Cb9xODVkg/wyClw\nLiHWlmH/eyC8nkiLYJKmV7566VWCV+gy+hC/DRstVVjIMG6LsqaPq6ycm7N8EV8s\nBb5uXIVHW6w5a20c40+W9G4EDYiQjdgEaf0FoMAWGDnOEaPsvjQk2/z5\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIDPDCCAiQCCQDxA75ydLHVoTANBgkqhkiG9w0BAQsFADBgMQswCQYDVQQGEwJS\nVTEPMA0GA1UECAwGTU9TQ09XMQ8wDQYDVQQHDAZNT1NDT1cxFTATBgNVBAoMDElO\nVEVSTUVESUFURTEYMBYGA1UEAwwPSU5URVJNRURJQVRFIENBMB4XDTIxMDczMDE1\nMTIyMloXDTI0MDUxOTE1MTIyMlowYDELMAkGA1UEBhMCUlUxDzANBgNVBAgMBk1P\nU0NPVzEPMA0GA1UEBwwGTU9TQ09XMRUwEwYDVQQKDAxJTlRFUk1FRElBVEUxGDAW\nBgNVBAMMD0lOVEVSTUVESUFURSBDQTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC\nAQoCggEBAKOrWdDVekCEf5r7KrQCCM007gTK2UeTk8Cr3Bp4XuBlQ8MyfzoBFpII\nSwMzqI3Cb5H8yTrVsDl6sIKYZLHPu/r8CD0z7d4lo20FM1yvKRS5I8azvKcf0MT7\nlK6flOg01b74EHrIzGfY0EVBSjognLtLLdTeUPwiUeFTicbWKC8tWVJbAAYcCfur\na5KDhUy0T5TFF9RCBue42AKLR70BaHPFDcdys7ybCSeekp4lwiJHrutSDrfGjlx1\nPt0y2IWZz4j99dXDG1wNCd7xj0WcM9uQWYxdyBf0oV5X88++hY1lEwE4XNPhwPJc\n8ed8Q+P5/BzfrBrmxsfU7y8mYphv1BsCAwEAATANBgkqhkiG9w0BAQsFAAOCAQEA\ngOHvrh66+bQoG3Lo8bfp7D1Xvm/Md3gJq2nMotl2BH1TvNzMV93fCXygRX8J8rTL\n7xjUC2SbOrFDWFq2hNJQagdecAeuG+U55BY6Wi8SsHw+fhgxQyl9wtXWwotQPmsD\nuRhR1rL3vEphgPLbxNBzA7Lvj+P89Ar988Qy+o5AiUzHMUuqZbGOqs8UcKCQP7e/\nIX+zqqFwqyI8f90SVySGgs574jo8jQFy3l5fnp6yK0MPWg2cBCjpa5H1A+5DADF+\nnryV6Ie/m/wfxmitZZN+YCJu+8Bmmdl/FCwbmiH+HCLhrO8gonH3K21cQujMyFF5\nc7OFj86hvhqbr4kzz1J8lg==\n-----END CERTIFICATE-----"

	terraformVersion = "0.12+compatible"

	ImagesPoint = "images"
)

type VarName string

//lint:ignore
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
	EC_PERMANENT_TOKEN_VAR    VarName = "EC_PERMANENT_TOKEN"
)

func getEnv(name VarName) string {
	return os.Getenv(string(name))
}

//revive:disable
var (
	EC_USERNAME           = getEnv(EC_USERNAME_VAR)
	EC_PASSWORD           = getEnv(EC_PASSWORD_VAR)
	EC_PERMANENT_TOKEN    = getEnv(EC_PERMANENT_TOKEN_VAR)
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

//nolint:unused
var varsMap = map[VarName]string{
	EC_USERNAME_VAR:           EC_USERNAME,
	EC_PASSWORD_VAR:           EC_PASSWORD,
	EC_PERMANENT_TOKEN_VAR:    EC_PERMANENT_TOKEN,
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

//nolint:unused
func testAccPreCheckVars(t *testing.T, vars ...VarName) {
	t.Helper()
	for _, name := range vars {
		if val := varsMap[name]; val == "" {
			t.Fatalf("'%s' must be set for acceptance test", name)
		}
	}
}

//nolint:unused
func testAccPreCheck(t *testing.T) {
	t.Helper()
	vars := map[string]interface{}{
		"EC_USERNAME":        EC_USERNAME,
		"EC_PASSWORD":        EC_PASSWORD,
		"EC_PERMANENT_TOKEN": EC_PERMANENT_TOKEN,
	}
	for k, v := range vars {
		if v == "" {
			t.Fatalf("'%s' must be set for acceptance test", k)
		}
	}
	checkNameAndID(t, "PROJECT")
	checkNameAndID(t, "REGION")
}

//nolint:unused
func checkNameAndID(t *testing.T, resourceType string) {
	// resourceType is a word in capital letters
	t.Helper()
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

//nolint:unused
func regionInfo() string {
	return objectInfo("REGION")
}

//nolint:unused
func projectInfo() string {
	return objectInfo("PROJECT")
}

//nolint:unused
func objectInfo(resourceType string) string {
	// resourceType is a word in capital letters
	keyID := fmt.Sprintf("TEST_%s_ID", resourceType)
	keyName := fmt.Sprintf("TEST_%s_NAME", resourceType)
	if objectID, exists := os.LookupEnv(keyID); exists {
		return fmt.Sprintf(`%s_id = %s`, strings.ToLower(resourceType), objectID)
	}
	return fmt.Sprintf(`%s_name = "%s"`, strings.ToLower(resourceType), os.Getenv(keyName))
}

//nolint:unused
func createTestClient(provider *edgecloud.ProviderClient, endpoint, version string) (*edgecloud.ServiceClient, error) {
	projectID := 0
	var err error
	if strProjectID, exists := os.LookupEnv("TEST_PROJECT_ID"); exists {
		projectID, err = strconv.Atoi(strProjectID)
		if err != nil {
			return nil, err
		}
	} else {
		projectID, err = edgecenter.GetProject(provider, 0, os.Getenv("TEST_PROJECT_NAME"))
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
		regionID, err = edgecenter.GetProject(provider, 0, os.Getenv("TEST_REGION_NAME"))
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

//nolint:unused
func getRegionIDAndProjectID() (int, int, error) {
	projectID := 0
	regionID := 0
	var err error
	if strProjectID, exists := os.LookupEnv("TEST_PROJECT_ID"); exists {
		projectID, err = strconv.Atoi(strProjectID)
		if err != nil {
			return 0, 0, err
		}
	}
	if strRegionID, exists := os.LookupEnv("TEST_REGION_ID"); exists {
		regionID, err = strconv.Atoi(strRegionID)
		if err != nil {
			return 0, 0, err
		}
	}

	return regionID, projectID, nil
}

//nolint:unused
func createTestConfig() (*edgecenter.Config, error) {
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

	userAgent := fmt.Sprintf("terraform/%s", terraformVersion)
	cloudClient, err := edgecloudV2.NewWithRetries(nil,
		edgecloudV2.SetUserAgent(userAgent),
		edgecloudV2.SetAPIKey(os.Getenv("EC_PERMANENT_TOKEN")),
		edgecloudV2.SetBaseURL(os.Getenv("EC_API")),
	)
	if err != nil {
		return nil, err
	}
	cloudClient.Region, cloudClient.Project, err = getRegionIDAndProjectID()

	if err != nil {
		return nil, err
	}

	cdnProvider := eccdnProvider.NewClient(EC_CDN_URL, eccdnProvider.WithSignerFunc(func(req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+provider.AccessToken())
		return nil
	}))
	cdnService := cdn.NewService(cdnProvider)

	storageAPI := EC_STORAGE_API
	stHost, stPath, err := edgecenter.ExtractHostAndPath(storageAPI)
	var storageClient *storageSDK.SDK
	if err == nil {
		storageClient = storageSDK.NewSDK(stHost, stPath, storageSDK.WithBearerAuth(provider.AccessToken))
	}

	var dnsClient *dnssdk.Client
	if EC_DNS_API != "" {
		baseURL, err := url.Parse(EC_DNS_API)
		if err == nil {
			authorizer := dnssdk.BearerAuth(provider.AccessToken())
			dnsClient = dnssdk.NewClient(authorizer, func(client *dnssdk.Client) {
				client.BaseURL = baseURL
			})
		}
	}

	config := edgecenter.Config{
		Provider:      provider,
		CloudClient:   cloudClient,
		CDNClient:     cdnService,
		StorageClient: storageClient,
		DNSClient:     dnsClient,
	}

	return &config, nil
}

//nolint:unused
func testAccCheckResourceExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// retrieve the resource by name from state
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("widget ID is not set")
		}
		return nil
	}
}

var (
	testAccProvider  *schema.Provider
	testAccProviders map[string]func() (*schema.Provider, error)
)

func TestMain(m *testing.M) {
	testAccProvider = edgecenter.Provider()
	testAccProviders = map[string]func() (*schema.Provider, error){
		"edgecenter": func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}
