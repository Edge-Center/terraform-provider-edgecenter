//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

// testAccCheckListNotEmpty verifies that a list attribute count is not zero
func testAccCheckListNotEmpty(resourceName, attr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		v, ok := rs.Primary.Attributes[attr]
		if !ok {
			return fmt.Errorf("attribute %s not set", attr)
		}
		if v == "0" || v == "" {
			return fmt.Errorf("attribute %s is empty (%s)", attr, v)
		}
		return nil
	}
}

// testAccCheckAllFlavorsType ensures all returned flavors have expected type
func testAccCheckAllFlavorsType(resourceName, expectedType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		countStr, ok := rs.Primary.Attributes[edgecenter.FlavorsField+".#"]
		if !ok {
			return fmt.Errorf("%s count not found", edgecenter.FlavorsField)
		}
		count, err := strconv.Atoi(countStr)
		if err != nil {
			return fmt.Errorf("failed to parse %s count: %w", edgecenter.FlavorsField, err)
		}
		if count == 0 {
			return fmt.Errorf("no flavors returned to validate type")
		}

		for i := 0; i < count; i++ {
			key := fmt.Sprintf("%s.%d.%s", edgecenter.FlavorsField, i, edgecenter.TypeField)
			tVal, ok := rs.Primary.Attributes[key]
			if !ok {
				return fmt.Errorf("attribute %s not found", key)
			}
			if tVal != expectedType {
				return fmt.Errorf("unexpected flavor type at index %d: got %s, expected %s", i, tVal, expectedType)
			}
		}
		return nil
	}
}

func TestAccFlavorDataSource_TypeFilter(t *testing.T) {
	t.Parallel()

	resourceName := "data.edgecenter_flavor.acctest"
	tpl := func(extra string) string {
		return fmt.Sprintf(`
            data "edgecenter_flavor" "acctest" {
              %s
              %s
              %s
            }
        `, projectInfo(), regionInfo(), extra)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				// No type filter: expect non-empty flavor list
				Config: tpl(""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					testAccCheckListNotEmpty(resourceName, edgecenter.FlavorsField+".#"),
				),
			},
			{
				Config: tpl("type = \"instance\""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					testAccCheckListNotEmpty(resourceName, edgecenter.FlavorsField+".#"),
					testAccCheckAllFlavorsType(resourceName, "instance"),
				),
			},
			{
				Config: tpl("type = \"baremetal\""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					testAccCheckListNotEmpty(resourceName, edgecenter.FlavorsField+".#"),
					testAccCheckAllFlavorsType(resourceName, "baremetal"),
				),
			},
			{
				Config: tpl("type = \"load_balancer\""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					testAccCheckListNotEmpty(resourceName, edgecenter.FlavorsField+".#"),
					testAccCheckAllFlavorsType(resourceName, "load_balancer"),
				),
			},
		},
	})
}

// TestAccFlavorDataSource_OptionsParams verifies that optional parameters are accepted without errors.
// Full result validation is not possible since the returned data depends on the current flavor catalog
func TestAccFlavorDataSource_OptionsParams(t *testing.T) {
	t.Parallel()

	resourceName := "data.edgecenter_flavor.acctest"
	tpl := func(extra string) string {
		return fmt.Sprintf(`
            data "edgecenter_flavor" "acctest" {
              %s
              %s
              %s
            }
        `, projectInfo(), regionInfo(), extra)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Include disabled flavors
				Config: tpl("include_disabled = true"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					testAccCheckListNotEmpty(resourceName, edgecenter.FlavorsField+".#"),
				),
			},
			{
				// Exclude windows-dedicated flavors
				Config: tpl("exclude_windows = true"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					testAccCheckListNotEmpty(resourceName, edgecenter.FlavorsField+".#"),
				),
			},
			{
				// Include prices in response
				Config: tpl("include_prices = true"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					testAccCheckListNotEmpty(resourceName, edgecenter.FlavorsField+".#"),
					// Verify that price fields are present in the response structure.
					// We check for empty strings rather than specific values since actual prices
					// may change over time or may not be present, making them unreliable for testing.
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.%s", edgecenter.FlavorsField, edgecenter.CurrencyCodeField), ""),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.%s", edgecenter.FlavorsField, edgecenter.PricePerHourField), "0"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.%s", edgecenter.FlavorsField, edgecenter.PricePerMonthField), "0"),
				),
			},
		},
	})
}
