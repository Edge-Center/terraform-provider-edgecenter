package platform

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	ProjectResource      = "edgecenter_project"
	ProjectDeleteTimeout = 1200 * time.Second
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProjectCreate,
		ReadContext:   resourceProjectRead,
		UpdateContext: resourceProjectUpdate,
		DeleteContext: resourceProjectDelete,
		Description:   `A project is a structural unit that helps manage and organize cloud resources`,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			edgecenter.IDField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Project ID.",
			},
			edgecenter.ClientIDField: {
				Type:        schema.TypeInt,
				Description: "The ID of the client.",
				Computed:    true,
				Optional:    true,
			},
			edgecenter.NameField: {
				Type:        schema.TypeString,
				Description: "Displayed project name.",
				Required:    true,
			},
			edgecenter.DescriptionField: {
				Type:        schema.TypeString,
				Description: "The description of the project.",
				Optional:    true,
			},
			edgecenter.StateField: {
				Type:        schema.TypeString,
				Description: "The state of the project.",
				Computed:    true,
			},
			edgecenter.CreatedAtField: {
				Type:        schema.TypeString,
				Description: "The datetime of the project creation. It is automatically generated when the project is created.",
				Computed:    true,
			},
			edgecenter.IsDefaultField: {
				Type:        schema.TypeBool,
				Description: "The default flag. There is always one default project for each client.",
				Computed:    true,
			},
		},
	}
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Project creating")

	clientConf := edgecenter.CloudClientConf{
		DoNotUseRegionID:  true,
		DoNotUseProjectID: true,
	}
	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, &clientConf)
	if err != nil {
		return diag.FromErr(err)
	}

	opts := &edgecloudV2.ProjectCreateRequest{
		Name:        d.Get(edgecenter.NameField).(string),
		Description: d.Get(edgecenter.DescriptionField).(string),
		ClientID:    strconv.Itoa(d.Get(edgecenter.ClientIDField).(int)),
	}
	log.Printf("Create project ops: %+v", opts)

	p, _, err := clientV2.Projects.Create(ctx, opts)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Project id (%d)", p.ID)

	d.SetId(strconv.Itoa(p.ID))

	log.Printf("[DEBUG] Finish Project creating (%d)", p.ID)

	return resourceProjectRead(ctx, d, m)
}

func resourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start FloatingIP reading")
	clientConf := edgecenter.CloudClientConf{
		DoNotUseRegionID:  true,
		DoNotUseProjectID: true,
	}
	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, &clientConf)
	if err != nil {
		return diag.FromErr(err)
	}

	project, response, err := clientV2.Projects.Get(ctx, d.Id())
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] Removing project %s because resource doesn't exist anymore", d.Id())
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] Retrieved Project %s: %#v", d.Id(), project)
	d.Set(edgecenter.ClientIDField, project.ClientID)
	d.Set(edgecenter.DescriptionField, project.Description)
	d.Set(edgecenter.StateField, project.State)
	d.Set(edgecenter.CreatedAtField, project.CreatedAt)
	d.Set(edgecenter.IsDefaultField, project.IsDefault)

	return nil
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Project updating")
	clientConf := edgecenter.CloudClientConf{
		DoNotUseRegionID:  true,
		DoNotUseProjectID: true,
	}
	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, &clientConf)
	if err != nil {
		return diag.FromErr(err)
	}

	updateOpts := edgecloudV2.ProjectUpdateRequest{}

	updateOpts.Name = d.Get(edgecenter.NameField).(string)
	updateOpts.Description = d.Get(edgecenter.DescriptionField).(string)

	_, _, err = clientV2.Projects.Update(ctx, d.Id(), &updateOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceProjectRead(ctx, d, m)
}

func resourceProjectDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Project deleting")
	var diags diag.Diagnostics
	clientConf := edgecenter.CloudClientConf{
		DoNotUseRegionID:  true,
		DoNotUseProjectID: true,
	}
	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, &clientConf)
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()

	results, _, err := clientV2.Projects.Delete(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, ProjectDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete project with ID: %s", id)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of Project deleting")

	return diags
}
