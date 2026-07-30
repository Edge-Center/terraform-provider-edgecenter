//go:build dns

package edgecenter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dns"
)

func TestAccDnsZoneRecord(t *testing.T) {
	t.Parallel()
	random := time.Now().Nanosecond()
	domain := "terraformtest"
	subDomain := fmt.Sprintf("key%d", random)
	name := fmt.Sprintf("%s_%s", subDomain, domain)
	zone := fmt.Sprintf("%s%d.com", domain, random)
	fullDomain := subDomain + "." + zone

	resourceName := fmt.Sprintf("%s.%s", dns.DNSZoneRecordResource, name)

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
		`, zone, dns.DNSZoneRecordResource, name, fullDomain)
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
		`, zone, dns.DNSZoneRecordResource, name, fullDomain)
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
					resource.TestCheckResourceAttr(resourceName, dns.DNSZoneRecordSchemaDomain, fullDomain),
					resource.TestCheckResourceAttr(resourceName, dns.DNSZoneRecordSchemaType, "TXT"),
					resource.TestCheckResourceAttr(resourceName, dns.DNSZoneRecordSchemaTTL, "60"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", dns.DNSZoneRecordSchemaFilter, dns.DNSZoneRecordSchemaFilterType),
						"geodistance"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", dns.DNSZoneRecordSchemaFilter, dns.DNSZoneRecordSchemaFilterLimit),
						"1"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", dns.DNSZoneRecordSchemaFilter, dns.DNSZoneRecordSchemaFilterStrict),
						"true"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", dns.DNSZoneRecordSchemaResourceRecord, dns.DNSZoneRecordSchemaContent),
						"1234"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", dns.DNSZoneRecordSchemaResourceRecord, dns.DNSZoneRecordSchemaEnabled),
						"true"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaLatLong,
						),
						"52.367"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.1",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaLatLong,
						),
						"4.9041"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaAsn,
						),
						"12345"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaIP,
						),
						"1.1.1.1"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaNotes,
						),
						"notes"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaContinents,
						),
						"asia"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaCountries,
						),
						"ru"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaDefault,
						),
						"true"),
				),
			},
			{
				Config: templateUpdate(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, dns.DNSZoneRecordSchemaDomain, fullDomain),
					resource.TestCheckResourceAttr(resourceName, dns.DNSZoneRecordSchemaType, "TXT"),
					resource.TestCheckResourceAttr(resourceName, dns.DNSZoneRecordSchemaTTL, "120"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s", dns.DNSZoneRecordSchemaResourceRecord, dns.DNSZoneRecordSchemaContent),
						"12345"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaLatLong,
						),
						"52.367"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.1",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaLatLong,
						),
						"4.9041"),
					resource.TestCheckNoResourceAttr(resourceName, fmt.Sprintf("%s.0.%s.0.%s.0",
						dns.DNSZoneRecordSchemaResourceRecord,
						dns.DNSZoneRecordSchemaMeta,
						dns.DNSZoneRecordSchemaMetaAsn,
					)),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaIP,
						),
						"1.1.2.2"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaNotes,
						),
						"notes"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaContinents,
						),
						"asia"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaCountries,
						),
						"cn"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s",
							dns.DNSZoneRecordSchemaResourceRecord,
							dns.DNSZoneRecordSchemaMeta,
							dns.DNSZoneRecordSchemaMetaDefault,
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

	resourceName := fmt.Sprintf("%s.%s", dns.DNSZoneRecordResource, name)

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
		`, zone, dns.DNSZoneRecordResource, name, fullDomain)
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
					resource.TestCheckResourceAttr(resourceName, dns.DNSZoneRecordSchemaDomain, fullDomain),
					resource.TestCheckResourceAttr(resourceName, dns.DNSZoneRecordSchemaType, "DNAME"),
					resource.TestCheckResourceAttr(resourceName, dns.DNSZoneRecordSchemaTTL, "600"),
					resource.TestCheckResourceAttr(
						resourceName,
						fmt.Sprintf("%s.0.%s", dns.DNSZoneRecordSchemaResourceRecord, dns.DNSZoneRecordSchemaContent),
						"yandex.ru.",
					),
					resource.TestCheckResourceAttr(
						resourceName,
						fmt.Sprintf("%s.0.%s", dns.DNSZoneRecordSchemaResourceRecord, dns.DNSZoneRecordSchemaEnabled),
						"true",
					),
				),
			},
		},
	})
}
