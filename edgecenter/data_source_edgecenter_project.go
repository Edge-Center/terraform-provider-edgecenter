package edgecenter

import (
	"context"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceProject() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceProjectRead,
		Description: "Represent project data.",
		Schema: map[string]*schema.Schema{
			IDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "Project ID.",
				ExactlyOneOf: []string{IDField, NameField},
			},

			ClientIDField: {
				Type:        schema.TypeInt,
				Description: "The ID of the client.",
				Computed:    true,
				Optional:    true,
			},
			NameField: {
				Type:         schema.TypeString,
				Description:  "Displayed project name.",
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{IDField, NameField},
			},
			DescriptionField: {
				Type:        schema.TypeString,
				Description: "The description of the project.",
				Computed:    true,
			},
			StateField: {
				Type:        schema.TypeString,
				Description: "The state of the project.",
				Computed:    true,
			},
			CreatedAtField: {
				Type:        schema.TypeString,
				Description: "The datetime of the project creation. It is automatically generated when the project is created.",
				Computed:    true,
			},
			IsDefaultField: {
				Type:        schema.TypeBool,
				Description: "The default flag. There is always one default project for each client.",
				Computed:    true,
			},
		},
	}
}

func dataSourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Project reading")

	clientConf := CloudClientConf{
		DoNotUseRegionID:  true,
		DoNotUseProjectID: true,
	}
	clientV2, err := InitCloudClient(ctx, d, m, &clientConf)
	if err != nil {
		return diag.FromErr(err)
	}

	projectName := d.Get(NameField).(string)
	projectID := d.Get(IDField).(int)

	log.Printf("[DEBUG] project id = %d", projectID)

	project, err := GetProjectV2(ctx, clientV2, projectID, projectName)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(project.ID))
	d.Set(NameField, project.Name)
	d.Set(ClientIDField, project.ClientID)
	d.Set(DescriptionField, project.Description)
	d.Set(StateField, project.State)
	d.Set(CreatedAtField, project.CreatedAt)
	d.Set(IsDefaultField, project.IsDefault)

	log.Println("[DEBUG] Finish Project reading")

	return nil
}
