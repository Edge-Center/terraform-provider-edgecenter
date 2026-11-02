package websec

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/Edge-Center/edgecentercdn-go/websec/sourcegroups"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const maxSourceGroupWeight = 100

var (
	sourceHostnamePattern = regexp.MustCompile(`(?i)^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+([a-z][a-z-]{0,61}[a-z]|xn--[a-z0-9]{1,59})\.?$`)

	validateSourceAddress = stringValidator(
		isSourceAddress,
		"be a host name or an IPv4 address with an optional port, without a scheme or a path, up to 255 characters",
	)
)

func resourceWebsecSourceGroup() *schema.Resource {
	return &schema.Resource{
		Description: "WebSec source group: the origins that protected resources and their locations forward requests to.\n\n" +
			"Every attribute is updated in place; nothing forces a replacement.\n\n" +
			"A group that resources or locations still use is not deleted: the API would delete them together with the " +
			"group, so the provider stops and lists them. To point a resource at another group and delete the old group in " +
			"the same apply, keep `create_before_destroy` in the `lifecycle` of the old group from the start: Terraform takes " +
			"the flag from state, and with it the resource is updated before the old group is deleted. Without the flag, " +
			"move the resources to the new group in one apply and remove the old group in the next one. If a group is itself " +
			"replaced with `create_before_destroy`, the new one needs another name because names are unique within the " +
			"client.\n\n" +
			"Import with `<group_id>` or `resource:<resource_id>`. The group id is the number in `source_group` of a " +
			"resource, and `resource:<resource_id>` imports the group that `edgecenter_websec_resource` uses. A resource " +
			"moved from the old protection API keeps the id of its `edgecenter_protection_resource`, so the second form " +
			"needs only the id from the old state.",
		CreateContext: resourceWebsecSourceGroupCreate,
		ReadContext:   resourceWebsecSourceGroupRead,
		UpdateContext: resourceWebsecSourceGroupUpdate,
		DeleteContext: resourceWebsecSourceGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importWebsecSourceGroup,
		},
		CustomizeDiff: customdiff.All(
			validateUniqueSources,
			validateSourceGroupWeight,
		),
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.All(validation.StringLenBetween(1, 255), validateTrimmed),
				Description:  "Name of the group, up to 255 characters, unique within the client.",
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validateTrimmed,
				Description:  "Description of the group.",
			},
			"use_next": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Pass a request on to the next source when the current one fails. Defaults to `true`.",
			},
			"source": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Description: "Sources of the group, kept in the order given. More than one source works only when the multiple " +
					"sources feature is enabled for the client.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateSourceAddress,
							Description: "Host name or IPv4 address of the origin with an optional port, for example `203.0.113.10` " +
								"or `origin.example.com:8080`, without a scheme or a path. The API also checks that a host name " +
								"resolves and that the origin is not in a private network.",
						},
						"mode": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  string(sourcegroups.SourceModePrimary),
							ValidateFunc: validation.StringInSlice([]string{
								string(sourcegroups.SourceModePrimary),
								string(sourcegroups.SourceModeBackup),
								string(sourcegroups.SourceModeDown),
							}, false),
							Description: "`primary`, `backup` (gets requests only when the primary sources fail) or `down` (gets " +
								"no requests). Defaults to `primary`.",
						},
						"weight": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      50,
							ValidateFunc: validation.IntBetween(1, 100),
							Description: "Weight of the source in load balancing, from 1 to 100. The weights of the primary sources " +
								"of the group add up to at most 100, so a third primary source with the default weight does not " +
								"fit. Defaults to `50`.",
						},
						"max_fails": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validation.IntAtLeast(1),
							Description: "Number of failed attempts within `fail_timeout` after which the source is considered " +
								"unavailable, at least 1. Defaults to `1`.",
						},
						"fail_timeout": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validation.IntAtLeast(1),
							Description: "Time in seconds in which `max_fails` failed attempts make the source unavailable, and for " +
								"which it then stays unavailable, at least 1. Defaults to `1`.",
						},
						"comment": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "",
							ValidateFunc: validation.All(validation.StringLenBetween(0, 255), validateTrimmed),
							Description:  "Comment on the source, up to 255 characters.",
						},
					},
				},
			},
		},
	}
}

func importWebsecSourceGroup(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	id, isResourceRef, err := parseIDOrResourceRef(d.Id())
	if err != nil {
		return nil, fmt.Errorf("import id %q must be <group_id> or resource:<resource_id>: %w", d.Id(), err)
	}

	if isResourceRef {
		resource, err := m.(*edgecenter.Config).WebsecClient.Resources().Get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("read resource %d: %w", id, err)
		}
		if resource.SourceGroup == nil {
			return nil, fmt.Errorf("resource %d has no source group", id)
		}
		id = *resource.SourceGroup
	}

	d.SetId(strconv.FormatInt(id, 10))

	return []*schema.ResourceData{d}, nil
}

func resourceWebsecSourceGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start WebSec source group creating")
	client := m.(*edgecenter.Config).WebsecClient

	group, err := client.SourceGroups().Create(ctx, expandSourceGroup(d))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(group.ID, 10))
	log.Printf("[DEBUG] Finish WebSec source group creating (id=%d)\n", group.ID)

	return resourceWebsecSourceGroupRead(ctx, d, m)
}

func resourceWebsecSourceGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec source group reading (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	group, err := client.SourceGroups().Get(ctx, id)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] WebSec source group %d not found, removing it from state\n", id)
			d.SetId("")

			return nil
		}

		return diag.FromErr(err)
	}

	d.Set("name", group.Name)
	d.Set("description", group.Description)
	d.Set("use_next", group.UseNext)
	if err := d.Set("source", flattenSources(group.Sources)); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Finish WebSec source group reading (id=%d)\n", id)

	return nil
}

func resourceWebsecSourceGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec source group updating (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if _, err := client.SourceGroups().Update(ctx, id, expandSourceGroup(d)); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Finish WebSec source group updating (id=%d)\n", id)

	return resourceWebsecSourceGroupRead(ctx, d, m)
}

func resourceWebsecSourceGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec source group deleting (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	group, err := client.SourceGroups().Get(ctx, id)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] WebSec source group %d is already deleted\n", id)
			d.SetId("")

			return nil
		}

		return diag.FromErr(err)
	}

	if len(group.RelatedResources) > 0 {
		return diag.Errorf("source group %d is still used by %s: point them to another source group or delete them "+
			"first, because the API deletes a source group together with the resources and locations that use it",
			id, describeSourceGroupUsers(group.RelatedResources))
	}

	if err := client.SourceGroups().Delete(ctx, id); err != nil && !isNotFound(err) {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish WebSec source group deleting (id=%d)\n", id)

	return nil
}

func expandSourceGroup(d *schema.ResourceData) *sourcegroups.GroupRequest {
	useNext := d.Get("use_next").(bool)

	return &sourcegroups.GroupRequest{
		Name:        d.Get("name").(string),
		UseNext:     &useNext,
		Description: d.Get("description").(string),
		Sources:     expandSources(d.Get("source").([]interface{})),
	}
}

func expandSources(items []interface{}) []sourcegroups.Source {
	sources := make([]sourcegroups.Source, 0, len(items))
	for _, item := range items {
		fields := item.(map[string]interface{})
		sources = append(sources, sourcegroups.Source{
			Source:      fields["source"].(string),
			Mode:        sourcegroups.SourceMode(fields["mode"].(string)),
			Weight:      fields["weight"].(int),
			MaxFails:    fields["max_fails"].(int),
			FailTimeout: fields["fail_timeout"].(int),
			Comment:     fields["comment"].(string),
		})
	}

	return sources
}

func flattenSources(sources []sourcegroups.Source) []interface{} {
	items := make([]interface{}, 0, len(sources))
	for _, source := range sources {
		items = append(items, map[string]interface{}{
			"source":       source.Source,
			"mode":         string(source.Mode),
			"weight":       source.Weight,
			"max_fails":    source.MaxFails,
			"fail_timeout": source.FailTimeout,
			"comment":      source.Comment,
		})
	}

	return items
}

func describeSourceGroupUsers(related []sourcegroups.RelatedResource) string {
	users := make([]string, 0, len(related))
	for _, resource := range related {
		user := fmt.Sprintf("resource %d (%s)", resource.ID, resource.Domain)
		if len(resource.Locations) > 0 {
			locations := make([]string, 0, len(resource.Locations))
			for _, location := range resource.Locations {
				locations = append(locations, fmt.Sprintf("%d (%s)", location.ID, location.Name))
			}
			user += " with locations " + strings.Join(locations, ", ")
		}
		users = append(users, user)
	}

	return strings.Join(users, "; ")
}

func validateUniqueSources(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	sources, known := knownSources(d, "source")
	if !known {
		return nil
	}

	seen := make(map[string]bool, len(sources))
	for _, item := range sources {
		source := item.(map[string]interface{})["source"].(string)
		if seen[source] {
			return fmt.Errorf("source %q is listed more than once, but sources of a group must be unique", source)
		}
		seen[source] = true
	}

	return nil
}

func validateSourceGroupWeight(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	sources, known := knownSources(d, "weight")
	if !known {
		return nil
	}

	total := 0
	for _, item := range sources {
		source := item.(map[string]interface{})
		if source["mode"].(string) == string(sourcegroups.SourceModePrimary) {
			total += source["weight"].(int)
		}
	}

	if total > maxSourceGroupWeight {
		return fmt.Errorf("the weights of primary sources add up to %d, but a source group allows at most %d", total, maxSourceGroupWeight)
	}

	return nil
}

func knownSources(d *schema.ResourceDiff, field string) ([]interface{}, bool) {
	if !d.NewValueKnown("source") {
		return nil, false
	}

	sources := d.Get("source").([]interface{})
	for i := range sources {
		if !d.NewValueKnown(fmt.Sprintf("source.%d.%s", i, field)) {
			return nil, false
		}
	}

	return sources, true
}

func isSourceAddress(value string) bool {
	host, port, hasPort := strings.Cut(value, ":")

	return len(value) <= 255 && isSourceHost(host) && (!hasPort || isSourcePort(port))
}

func isSourceHost(host string) bool {
	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}

	return len(host) <= 253 && sourceHostnamePattern.MatchString(host)
}

func isSourcePort(port string) bool {
	number, err := strconv.Atoi(port)

	return err == nil && port[0] >= '1' && port[0] <= '9' && number <= 65535
}
