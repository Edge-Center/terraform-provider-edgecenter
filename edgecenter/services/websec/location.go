package websec

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/AlekSi/pointer"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/Edge-Center/edgecentercdn-go/websec/locations"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

var validateLocationPattern = stringValidator(
	isLocationPattern,
	"start with \"/\" or \"^/\", hold more than slashes, have no whitespace, quotes or semicolons and be up to 255 characters",
)

func resourceWebsecLocation() *schema.Resource {
	return &schema.Resource{
		Description: "WebSec location: a rule of an `edgecenter_websec_resource` for the requests whose path matches `location`. " +
			"These requests can go to their own source group and get their own protection options. `name` and `location` are " +
			"unique within the resource. The API refuses to create, change or delete the locations of a resource that the " +
			"administrator has disabled (its `enabled` is `false`).\n\n" +
			"Changing `resource_id` replaces the location; every other attribute is updated in place.\n\n" +
			"Import with `<resource_id>:<location_id>`. `GET <edgecenter_websec_api>/resources/<resource_id>/locations` " +
			"lists the locations of a resource with their ids, and `edgecenter_websec_api` defaults to " +
			"`<api_endpoint>/v2/protection`.",
		CreateContext: resourceWebsecLocationCreate,
		ReadContext:   resourceWebsecLocationRead,
		UpdateContext: resourceWebsecLocationUpdate,
		DeleteContext: resourceWebsecLocationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importWebsecLocation,
		},
		CustomizeDiff: validateOptionsNotEmpty,
		Schema: map[string]*schema.Schema{
			"resource_id": {
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "ID of the `edgecenter_websec_resource` the location belongs to. Changing it replaces the location.",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.All(validation.StringLenBetween(1, 255), validateTrimmed),
				Description:  "Name of the location, up to 255 characters, unique within the resource.",
			},
			"location": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateLocationPattern,
				Description: "Regular expression for the request path that triggers the location. It starts with \"/\" or " +
					"\"^/\", holds more than slashes, has no whitespace, quotes or semicolons and is up to 255 characters. The " +
					"API also checks that it is a valid PCRE pattern. Unique within the resource.",
			},
			"active": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enable or disable the location. Defaults to `true`.",
			},
			"weight": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      1,
				ValidateFunc: validation.IntBetween(1, math.MaxInt32),
				Description:  "Weight of the location, from 1 to 2147483647. Defaults to `1`.",
			},
			"source_group": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description: "ID of the `edgecenter_websec_source_group` the requests of the location go to. Without it they " +
					"go to the source group of the resource.",
			},
			"options": locationOptionsSchema(),
		},
	}
}

func isLocationPattern(value string) bool {
	path, _ := strings.CutPrefix(value, "^")

	return len(value) <= 255 &&
		strings.HasPrefix(path, "/") &&
		strings.Trim(path, "/") != "" &&
		!strings.ContainsAny(value, `'";`) &&
		!strings.ContainsFunc(value, unicode.IsSpace)
}

func importWebsecLocation(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	rawResourceID, rawLocationID, _ := strings.Cut(d.Id(), ":")

	resourceID, err := parseID(rawResourceID)
	if err != nil {
		return nil, fmt.Errorf("import id %q must be <resource_id>:<location_id>: %w", d.Id(), err)
	}

	locationID, err := parseID(rawLocationID)
	if err != nil {
		return nil, fmt.Errorf("import id %q must be <resource_id>:<location_id>: %w", d.Id(), err)
	}

	d.SetId(strconv.FormatInt(locationID, 10))
	if err := d.Set("resource_id", resourceID); err != nil {
		return nil, fmt.Errorf("set resource_id: %w", err)
	}

	return []*schema.ResourceData{d}, nil
}

func expandLocationSourceGroup(d *schema.ResourceData) *int64 {
	if group, ok := d.GetOk("source_group"); ok {
		return pointer.ToInt64(int64(group.(int)))
	}

	return nil
}

func resourceWebsecLocationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start WebSec location creating")
	client := m.(*edgecenter.Config).WebsecClient

	location, err := client.Locations().Create(ctx, int64(d.Get("resource_id").(int)), &locations.CreateRequest{
		Name:        d.Get("name").(string),
		Location:    d.Get("location").(string),
		Active:      pointer.ToBool(d.Get("active").(bool)),
		SourceGroup: expandLocationSourceGroup(d),
		Weight:      pointer.ToInt(d.Get("weight").(int)),
		Options:     expandLocationOptions(d, false),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(location.ID, 10))
	log.Printf("[DEBUG] Finish WebSec location creating (id=%d)\n", location.ID)

	return resourceWebsecLocationRead(ctx, d, m)
}

func resourceWebsecLocationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec location reading (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	resourceID := int64(d.Get("resource_id").(int))

	location, err := client.Locations().Get(ctx, resourceID, id)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] WebSec location %d of resource %d not found, removing it from state\n", id, resourceID)
			d.SetId("")

			return nil
		}

		return diag.FromErr(err)
	}

	locationOptions, err := flattenLocationOptions(location.Options)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", location.Name)
	d.Set("location", location.Location)
	d.Set("active", location.Active)
	d.Set("weight", location.Weight)
	d.Set("source_group", pointer.GetInt64(location.SourceGroup))
	if err := d.Set("options", locationOptions); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Finish WebSec location reading (id=%d)\n", id)

	return nil
}

func resourceWebsecLocationUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec location updating (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := locations.UpdateRequest{
		Name:        d.Get("name").(string),
		Location:    d.Get("location").(string),
		Active:      d.Get("active").(bool),
		SourceGroup: expandLocationSourceGroup(d),
		Weight:      pointer.ToInt(d.Get("weight").(int)),
		Options:     expandLocationOptions(d, true),
	}
	if _, err := client.Locations().Update(ctx, int64(d.Get("resource_id").(int)), id, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Finish WebSec location updating (id=%d)\n", id)

	return resourceWebsecLocationRead(ctx, d, m)
}

func resourceWebsecLocationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec location deleting (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.Locations().Delete(ctx, int64(d.Get("resource_id").(int)), id); err != nil && !isNotFound(err) {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish WebSec location deleting (id=%d)\n", id)

	return nil
}
