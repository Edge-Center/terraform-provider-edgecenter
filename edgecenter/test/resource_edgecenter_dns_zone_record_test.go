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
	zone := domain + ".com"
	fullDomain := subDomain + "." + zone

	resourceName := fmt.Sprintf("%s.%s", edgecenter.DNSZoneRecordResource, name)

	templateCreate := func() string {
		return fmt.Sprintf(`
resource "%s" "%s" {
  zone = "%s"
  domain = "%s"
  type = "TXT"
  ttl = 10

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
    failover {
        frequency = 10
        port = 443
        protocol = "ICMP"
        timeout = 10
    }
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
	  countries = ["russia"]
	  default = true
  	}
  }
}
		`, edgecenter.DNSZoneRecordResource, name, zone, fullDomain)
	}
	templateUpdate := func() string {
		return fmt.Sprintf(`
resource "%s" "%s" {
  zone = "%s"
  domain = "%s"
  type = "TXT"
  ttl = 20

  resource_record {
    content  = "12345"
    
    meta {
      latlong = [52.367,4.9041]
	  ip = ["1.1.2.2"]
	  notes = ["notes"]
	  continents = ["america"]
	  countries = ["usa"]
	  default = false
  	}
  }
}
		`, edgecenter.DNSZoneRecordResource, name, zone, fullDomain)
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
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneRecordSchemaTTL, "10"),
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
						"russia"),
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
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneRecordSchemaTTL, "20"),
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
						"america"),
					resource.TestCheckResourceAttr(resourceName,
						fmt.Sprintf("%s.0.%s.0.%s.0",
							edgecenter.DNSZoneRecordSchemaResourceRecord,
							edgecenter.DNSZoneRecordSchemaMeta,
							edgecenter.DNSZoneRecordSchemaMetaCountries,
						),
						"usa"),
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
