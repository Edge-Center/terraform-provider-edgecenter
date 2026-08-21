package dbaas

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/shared/tfutil"
)

const DBaaSUserResource = "edgecenter_dbaas_user"

func resourceDBaaSUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDBaaSUserCreate,
		ReadContext:   resourceDBaaSUserRead,
		UpdateContext: resourceDBaaSUserUpdate,
		DeleteContext: resourceDBaaSUserDelete,
		Description:   "Represent DBaaS database user resource.",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, clusterID, username, err := tfutil.ImportStringParserExtended(d.Id())
				if err != nil {
					return nil, fmt.Errorf("importing DBaaS user: %w", err)
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.Set(edgecenter.DBaaSClusterIDField, clusterID)
				d.SetId(username)
				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			edgecenter.ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.DBaaSClusterIDField: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			edgecenter.NameField: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			edgecenter.PasswordField: {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			edgecenter.DBaaSUserDatabasesField: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceDBaaSUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS user creating")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(edgecenter.DBaaSClusterIDField).(string)
	name := d.Get(edgecenter.NameField).(string)

	createOpts := edgecloudV2.DBaaSUserCreateRequest{
		Name:     name,
		Password: d.Get(edgecenter.PasswordField).(string),
	}

	if v, ok := d.GetOk(edgecenter.DBaaSUserDatabasesField); ok {
		for _, db := range v.([]interface{}) {
			createOpts.Databases = append(createOpts.Databases, edgecloudV2.DBaaSUserDatabase{
				Name: db.(string),
			})
		}
	}

	_, _, err = clientV2.DBaaS.UserCreate(ctx, clusterID, createOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(name)
	tflog.Info(ctx, fmt.Sprintf("DBaaS user id = %s", d.Id()))

	return resourceDBaaSUserRead(ctx, d, m)
}

func resourceDBaaSUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS user reading")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(edgecenter.DBaaSClusterIDField).(string)
	username := d.Id()

	user, resp, err := clientV2.DBaaS.UserGet(ctx, clusterID, username)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, fmt.Sprintf("[WARN] Removing DBaaS user %s because resource doesn't exist anymore", d.Id()))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set(edgecenter.NameField, user.Name)

	databases := make([]string, len(user.Databases))
	for i, db := range user.Databases {
		databases[i] = db.Name
	}
	d.Set(edgecenter.DBaaSUserDatabasesField, databases)

	return diag.Diagnostics{}
}

func resourceDBaaSUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS user updating")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(edgecenter.DBaaSClusterIDField).(string)
	username := d.Id()

	if d.HasChange(edgecenter.PasswordField) {
		updateOpts := edgecloudV2.DBaaSUserUpdateRequest{
			Password: d.Get(edgecenter.PasswordField).(string),
		}
		_, err = clientV2.DBaaS.UserUpdate(ctx, clusterID, username, updateOpts)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(edgecenter.DBaaSUserDatabasesField) {
		oldRaw, newRaw := d.GetChange(edgecenter.DBaaSUserDatabasesField)
		oldList := oldRaw.([]interface{})
		newList := newRaw.([]interface{})

		oldDBs := make(map[string]bool)
		for _, v := range oldList {
			oldDBs[v.(string)] = true
		}
		newDBs := make(map[string]bool)
		for _, v := range newList {
			newDBs[v.(string)] = true
		}

		for db := range newDBs {
			if !oldDBs[db] {
				_, err = clientV2.DBaaS.UserGrantAccess(ctx, clusterID, username, db)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		}

		for db := range oldDBs {
			if !newDBs[db] {
				_, err = clientV2.DBaaS.UserRevokeAccess(ctx, clusterID, username, db)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	return resourceDBaaSUserRead(ctx, d, m)
}

func resourceDBaaSUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS user deleting")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(edgecenter.DBaaSClusterIDField).(string)
	username := d.Id()

	_, err = clientV2.DBaaS.UserDelete(ctx, clusterID, username)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	tflog.Info(ctx, "Finish of DBaaS user deleting")

	return diag.Diagnostics{}
}
