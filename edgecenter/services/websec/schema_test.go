package websec

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

func TestServiceRegistersEveryWebsecName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "websec", Service{}.Name())

	resources := Service{}.Resources()
	require.Len(t, resources, 5)
	for _, name := range []string{
		"edgecenter_websec_resource",
		"edgecenter_websec_source_group",
		"edgecenter_websec_ssl_certificate",
		"edgecenter_websec_location",
		"edgecenter_websec_lecert",
	} {
		require.Containsf(t, resources, name, "resource %q is not registered", name)
	}

	require.Nil(t, Service{}.DataSources())
}

func TestEveryWebsecAttributeIsEitherConfiguredOrComputed(t *testing.T) {
	t.Parallel()

	require.Empty(t, websecAttributesWhere(func(attribute *schema.Schema) bool {
		return attribute.Optional && attribute.Computed
	}))
}

func TestOnlyTheDomainAndTheParentLinksForceReplacement(t *testing.T) {
	t.Parallel()

	require.ElementsMatch(t, []string{
		"edgecenter_websec_resource.domain",
		"edgecenter_websec_location.resource_id",
		"edgecenter_websec_lecert.resource_id",
		"edgecenter_websec_lecert.cert_type",
	}, websecAttributesWhere(func(attribute *schema.Schema) bool { return attribute.ForceNew }))
}

func TestOnlyThePrivateKeyIsSensitive(t *testing.T) {
	t.Parallel()

	require.Equal(t, []string{"edgecenter_websec_ssl_certificate.private_key"},
		websecAttributesWhere(func(attribute *schema.Schema) bool { return attribute.Sensitive }))
}

func websecAttributesWhere(match func(*schema.Schema) bool) []string {
	var paths []string
	for name, resource := range (Service{}).Resources() {
		paths = appendAttributesWhere(paths, name, resource.Schema, match)
	}

	return paths
}

func appendAttributesWhere(paths []string, prefix string, attributes map[string]*schema.Schema, match func(*schema.Schema) bool) []string {
	for name, attribute := range attributes {
		path := prefix + "." + name
		if match(attribute) {
			paths = append(paths, path)
		}

		if elem, ok := attribute.Elem.(*schema.Resource); ok {
			paths = appendAttributesWhere(paths, path, elem.Schema, match)
		}
	}

	return paths
}
