//go:build dns

package edgecenter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccDnsZoneRecord(t *testing.T) {
	t.Parallel()
	random := time.Now().Nanosecond()
	domain := "terraformtest"
	subDomain := fmt.Sprintf("key%d", random)
	name := fmt.Sprintf("%s_%s", subDomain, domain)
	zone := fmt.Sprintf("%s%d.com", domain, random)
	fullDomain := subDomain + "." + zone

	resourceName := fmt.Sprintf("%s.%s", edgecenter.DNSZoneRecordResource, name)

	templateCreate := func() string {
		return fmt.Sprintf(`
resource "edgecenter_dns_zone" "acctest" {
  name = "%s"
}

resource "%s" "%s" {
  zone = edgecenter_dns_zone.acctest.name
  domain = "%s"
  type = "TXT"
  ttl = 60

  filter {
    type = "geodistance"
    limit = 1
    strict = true
  }

  filter {
    limit = 1
    type   = "is_healthy"
  }

  meta {
  }

  resource_record {
    content  = "1234"
    enabled = true
    
    meta {
      latlong = [52.367,4.9041]
	  asn = [12345]
	  ip = ["1.1.1.1"]
	  notes = ["notes"]
	  continents = ["asia"]
	  countries = ["ru"]
	  default = true
  	}
  }
}
		`, zone, edgecenter.DNSZoneRecordResource, name, fullDomain)
	}
	templateUpdate := func() string {
		return fmt.Sprintf(`
resource "edgecenter_dns_zone" "acctest" {
  name = "%s"
}

resource "%s" "%s" {
  zone = edgecenter_dns_zone.acctest.name
  domain = "%s"
  type = "TXT"
  ttl = 120

  meta {
  }

  resource_record {
    content  = "12345"
    
    meta {
      latlong = [52.367,4.9041]
	  ip = ["1.1.2.2"]
	  notes = ["notes"]
	  continents = ["asia"]
	  countries = ["cn"]
	  default = false
  	}
  }
}
		`, zone, edgecenter.DNSZoneRecordResource, name, fullDomain)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_DNS_URL_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: templateCreate(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneRecordSchemaDomain, fullDomain),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneRecordSchemaType, "TXT"),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneRecordSchemaTTL, "60"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", edgecenter.DNSZoneRecordSchemaFilter, edgecenter.DNSZoneRecordSchemaFilterType),
						"geodistance"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", edgecenter.DNSZoneRecordSchemaFilter, edgecenter.DNSZoneRecordSchemaFilterLimit),
						"1"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", edgecenter.DNSZoneRecordSchemaFilter, edgecenter.DNSZoneRecordSchemaFilterStrict),
						"true"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", edgecenter.DNSZoneRecordSchemaResourceRecord, edgecenter.DNSZoneRecordSchemaContent),
						"1234"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", edgecenter.DNSZoneRecordSchemaResourceRecord, edgecenter.DNSZoneRecordSchemaEnabled),
						"true"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaLatLong,
						),
						"52.367"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.1",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaLatLong,
						),
						"4.9041"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaAsn,
						),
						"12345"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaIP,
						),
						"1.1.1.1"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaNotes,
						),
						"notes"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaContinents,
						),
						"asia"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaCountries,
						),
						"ru"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaDefault,
						),
						"true"),
				),
			},
			{
				Config: templateUpdate(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneRecordSchemaDomain, fullDomain),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneRecordSchemaType, "TXT"),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneRecordSchemaTTL, "120"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", edgecenter.DNSZoneRecordSchemaResourceRecord, edgecenter.DNSZoneRecordSchemaContent),
						"12345"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaLatLong,
						),
						"52.367"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.1",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaLatLong,
						),
						"4.9041"),
					resource.TestCheckNoResourceAttr(resourceName, fmt.Sprintf("%s.0.%s.0.%s.0",
						edgecenter.DNSZoneRecordSchemaResourceRecord,
						edgecenter.DNSZoneRecordSchemaMeta,
						edgecenter.DNSZoneRecordSchemaMetaAsn,
					)),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaIP,
						),
						"1.1.2.2"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaNotes,
						),
						"notes"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaContinents,
						),
						"asia"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaCountries,
						),
						"cn"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaDefault,
						),
						"false"),
				),
			},
		},
	})
}

func TestAccDnsZoneRecordDNAME(t *testing.T) {
	t.Parallel()
	random := time.Now().Nanosecond()
	domain := "terraformtest"
	name := fmt.Sprintf("dname_%d", random)
	zone := fmt.Sprintf("%s%d.com", domain, random)
	fullDomain := fmt.Sprintf("dname%d.%s", random, zone)

	resourceName := fmt.Sprintf("%s.%s", edgecenter.DNSZoneRecordResource, name)

	templateCreate := func() string {
		return fmt.Sprintf(`
resource "edgecenter_dns_zone" "acctest" {
  name = "%s"
}

resource "%s" "%s" {
  zone = edgecenter_dns_zone.acctest.name
  domain = "%s"
  type = "DNAME"
  ttl = 600

  meta {
  }

  resource_record {
    content = "yandex.ru."
    enabled = true
  }
}
		`, zone, edgecenter.DNSZoneRecordResource, name, fullDomain)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_DNS_URL_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: templateCreate(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneRecordSchemaDomain, fullDomain),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneRecordSchemaType, "DNAME"),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneRecordSchemaTTL, "600"),
					resource.TestCheckResourceAttr(
						resourceName,
						fmt.Sprintf("%s.0.%s", edgecenter.DNSZoneRecordSchemaResourceRecord, edgecenter.DNSZoneRecordSchemaContent),
						"yandex.ru.",
					),
					resource.TestCheckResourceAttr(
						resourceName,
						fmt.Sprintf("%s.0.%s", edgecenter.DNSZoneRecordSchemaResourceRecord, edgecenter.DNSZoneRecordSchemaEnabled),
						"true",
					),
				),
			},
		},
	})
}
