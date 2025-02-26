package scaleway

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceScalewayCockpit() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceScalewayCockpit().Schema)

	dsSchema["project_id"] = &schema.Schema{
		Type:         schema.TypeString,
		Description:  "The project_id you want to attach the resource to",
		Optional:     true,
		ValidateFunc: validationUUID(),
	}
	delete(dsSchema, "plan")

	return &schema.Resource{
		ReadContext: dataSourceScalewayCockpitRead,
		Schema:      dsSchema,
	}
}

func dataSourceScalewayCockpitRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, err := cockpitAPI(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	projectID := d.Get("project_id").(string)

	res, err := waitForCockpit(ctx, api, projectID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(res.ProjectID)
	_ = d.Set("project_id", res.ProjectID)
	_ = d.Set("plan_id", res.Plan.ID)
	_ = d.Set("endpoints", flattenCockpitEndpoints(res.Endpoints))

	return nil
}
