package edgecenter

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercdn-go/origingroups"
)

func resourceCDNOriginGroup() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Add the source group name.",
			},
			"use_next": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Specify whether or not the CDN will use the next source in the list if your source responds with an HTTP status code of 4XX or 5XX.",
			},
			"origin": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "Add information about your sources.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Enter the source’s domain name or the IP address with a custom port (if any).",
						},
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "Enable or disable the source. The source group must contain at least one enabled source.",
						},
						"backup": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "If set to \"true\", this source will not be used until one of the active sources becomes unavailable.",
						},
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"authorization": {
				Type:        schema.TypeSet,
				MaxItems:    1,
				Optional:    true,
				Description: "Add information about authorization.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auth_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The type of authorization on the source. It can take two values - aws_signature_v2 or aws_signature_v4.",
						},
						"access_key_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Specify the access key ID in 20 alphanumeric characters.",
						},
						"secret_key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Specify the secret access key. The value must be between 32 and 40 characters and may include alphanumeric characters, slashes, pluses, hyphens, and underscores.",
						},
						"bucket_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Specify the bucket name. The name is restricted to 255 symbols and may include alphanumeric characters, slashes, pluses, hyphens, and underscores.",
						},
					},
				},
			},
			"consistent_balancing": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Consistent load balancing (consistent hashing) for the source group",
			},
		},
		CreateContext: resourceCDNOriginGroupCreate,
		ReadContext:   resourceCDNOriginGroupRead,
		UpdateContext: resourceCDNOriginGroupUpdate,
		DeleteContext: resourceCDNOriginGroupDelete,
		Description:   "Represent origin group",
	}
}

func resourceCDNOriginGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start CDN OriginGroup creating")
	config := m.(*Config)
	client := config.CDNClient

	var req origingroups.GroupRequest
	req.Name = d.Get("name").(string)
	req.UseNext = d.Get("use_next").(bool)
	req.Origins = setToOriginRequests(d.Get("origin").(*schema.Set))
	req.Authorization = setToAuthRequest(d.Get("authorization").(*schema.Set))
	req.ConsistentBalancing = d.Get("consistent_balancing").(bool)

	result, err := client.OriginGroups().Create(ctx, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", result.ID))
	resourceCDNOriginGroupRead(ctx, d, m)

	log.Printf("[DEBUG] Finish CDN OriginGroup creating (id=%d)\n", result.ID)

	return nil
}

func resourceCDNOriginGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	groupID := d.Id()
	log.Printf("[DEBUG] Start CDN OriginGroup reading (id=%s)\n", groupID)
	config := m.(*Config)
	client := config.CDNClient

	id, err := strconv.ParseInt(groupID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	result, err := client.OriginGroups().Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", result.Name)
	d.Set("use_next", result.UseNext)
	if err := d.Set("origin", originsToSet(result.Origins)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("authorization", authToSet(result.Authorization)); err != nil {
		return diag.FromErr(err)
	}
	d.Set("consistent_balancing", result.ConsistentBalancing)

	log.Println("[DEBUG] Finish CDN OriginGroup reading")

	return nil
}

func resourceCDNOriginGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	groupID := d.Id()
	log.Printf("[DEBUG] Start CDN OriginGroup updating (id=%s)\n", groupID)
	config := m.(*Config)
	client := config.CDNClient

	id, err := strconv.ParseInt(groupID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	var req origingroups.GroupRequest
	req.Name = d.Get("name").(string)
	req.UseNext = d.Get("use_next").(bool)
	req.Origins = setToOriginRequests(d.Get("origin").(*schema.Set))
	req.Authorization = setToAuthRequest(d.Get("authorization").(*schema.Set))
	req.ConsistentBalancing = d.Get("consistent_balancing").(bool)

	if _, err := client.OriginGroups().Update(ctx, id, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish CDN OriginGroup updating")

	return resourceCDNOriginGroupRead(ctx, d, m)
}

func resourceCDNOriginGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start CDN OriginGroup deleting (id=%s)\n", resourceID)

	config := m.(*Config)
	client := config.CDNClient

	id, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.OriginGroups().Delete(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish CDN Resource deleting")

	return nil
}

func setToOriginRequests(s *schema.Set) []origingroups.OriginRequest {
	origins := make([]origingroups.OriginRequest, 0)
	for _, fields := range s.List() {
		var originReq origingroups.OriginRequest

		for key, val := range fields.(map[string]interface{}) {
			switch key {
			case "source":
				originReq.Source = val.(string)
			case "enabled":
				originReq.Enabled = val.(bool)
			case "backup":
				originReq.Backup = val.(bool)
			}
		}

		origins = append(origins, originReq)
	}

	return origins
}

func originsToSet(origins []origingroups.Origin) *schema.Set {
	s := &schema.Set{F: originSetIDFunc}

	for _, origin := range origins {
		fields := make(map[string]interface{})
		fields["id"] = origin.ID
		fields["source"] = origin.Source
		fields["enabled"] = origin.Enabled
		fields["backup"] = origin.Backup

		s.Add(fields)
	}

	return s
}

func originSetIDFunc(i interface{}) int {
	fields := i.(map[string]interface{})
	h := md5.New()

	key := fmt.Sprintf("%d-%s-%t-%t", fields["id"], fields["source"], fields["enabled"], fields["backup"])
	log.Printf("[DEBUG] Origin Set ID = %s\n", key)

	io.WriteString(h, key)

	return int(binary.BigEndian.Uint64(h.Sum(nil)))
}

func setToAuthRequest(set *schema.Set) *origingroups.Authorization {
	if set.Len() == 0 {
		return nil
	}

	fields := set.List()[0].(map[string]interface{})
	return &origingroups.Authorization{
		AuthType:    fields["auth_type"].(string),
		AccessKeyID: fields["access_key_id"].(string),
		SecretKey:   fields["secret_key"].(string),
		BucketName:  fields["bucket_name"].(string),
	}
}

func authToSet(auth *origingroups.Authorization) *schema.Set {
	if auth == nil {
		return nil
	}

	return schema.NewSet(schema.HashResource(&schema.Resource{
		Schema: map[string]*schema.Schema{
			"auth_type": {
				Type: schema.TypeString,
			},
			"access_key_id": {
				Type: schema.TypeString,
			},
			"secret_key": {
				Type: schema.TypeString,
			},
			"bucket_name": {
				Type: schema.TypeString,
			},
		},
	}), []interface{}{
		map[string]interface{}{
			"auth_type":     auth.AuthType,
			"access_key_id": auth.AccessKeyID,
			"secret_key":    auth.SecretKey,
			"bucket_name":   auth.BucketName,
		},
	})
}
