package platform

import (
	"context"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const ProjectDataSource = "edgecenter_project"

func dataSourceProject() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceProjectRead,
		Description: "Represent project data.",
		Schema: map[string]*schema.Schema{
			edgecenter.IDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "Project ID.",
				ExactlyOneOf: []string{edgecenter.IDField, edgecenter.NameField},
			},

			edgecenter.ClientIDField: {
				Type:        schema.TypeInt,
				Description: "The ID of the client.",
				Computed:    true,
				Optional:    true,
			},
			edgecenter.NameField: {
				Type:         schema.TypeString,
				Description:  "Displayed project name.",
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{edgecenter.IDField, edgecenter.NameField},
			},
			edgecenter.DescriptionField: {
				Type:        schema.TypeString,
				Description: "The description of the project.",
				Computed:    true,
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

func dataSourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Project reading")

	clientConf := edgecenter.CloudClientConf{
		DoNotUseRegionID:  true,
		DoNotUseProjectID: true,
	}
	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, &clientConf)
	if err != nil {
		return diag.FromErr(err)
	}

	projectName := d.Get(edgecenter.NameField).(string)
	projectID := d.Get(edgecenter.IDField).(int)

	log.Printf("[DEBUG] project id = %d", projectID)

	project, err := edgecenter.GetProjectV2(ctx, clientV2, projectID, projectName)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(project.ID))
	d.Set(edgecenter.NameField, project.Name)
	d.Set(edgecenter.ClientIDField, project.ClientID)
	d.Set(edgecenter.DescriptionField, project.Description)
	d.Set(edgecenter.StateField, project.State)
	d.Set(edgecenter.CreatedAtField, project.CreatedAt)
	d.Set(edgecenter.IsDefaultField, project.IsDefault)

	log.Println("[DEBUG] Finish Project reading")

	return nil
}
