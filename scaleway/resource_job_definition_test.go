package scaleway

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	jobs "github.com/scaleway/scaleway-sdk-go/api/jobs/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_job_definition", &resource.Sweeper{
		Name: "scaleway_job_definition",
		F:    testSweepJobDefinition,
	})
}

func testSweepJobDefinition(_ string) error {
	return sweepRegions((&jobs.API{}).Regions(), func(scwClient *scw.Client, region scw.Region) error {
		jobsAPI := jobs.NewAPI(scwClient)
		l.Debugf("sweeper: destroying the jobs definitions in (%s)", region)
		listJobDefinitions, err := jobsAPI.ListJobDefinitions(
			&jobs.ListJobDefinitionsRequest{
				Region: region,
			}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing definition in (%s) in sweeper: %s", region, err)
		}

		for _, definition := range listJobDefinitions.JobDefinitions {
			err := jobsAPI.DeleteJobDefinition(&jobs.DeleteJobDefinitionRequest{
				JobDefinitionID: definition.ID,
				Region:          region,
			})
			if err != nil {
				l.Debugf("sweeper: error (%s)", err)

				return fmt.Errorf("error deleting definition in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayJobDefinition_Basic(t *testing.T) {
	tt := NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayJobDefinitionDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_job_definition main {
						name = "test-jobs-job-definition-basic"
						cpu_limit = 120
						memory_limit = 256
						image_uri = "docker.io/alpine:latest"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayJobDefinitionExists(tt, "scaleway_job_definition.main"),
					testCheckResourceAttrUUID("scaleway_job_definition.main", "id"),
					resource.TestCheckResourceAttr("scaleway_job_definition.main", "name", "test-jobs-job-definition-basic"),
				),
			},
		},
	})
}

func TestAccScalewayJobDefinition_Timeout(t *testing.T) {
	tt := NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayJobDefinitionDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_job_definition main {
						name = "test-jobs-job-definition-timeout"
						cpu_limit = 120
						memory_limit = 256
						image_uri = "docker.io/alpine:latest"
						timeout = "20m"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayJobDefinitionExists(tt, "scaleway_job_definition.main"),
					testCheckResourceAttrUUID("scaleway_job_definition.main", "id"),
					resource.TestCheckResourceAttr("scaleway_job_definition.main", "name", "test-jobs-job-definition-timeout"),
					resource.TestCheckResourceAttr("scaleway_job_definition.main", "timeout", "20m0s"),
				),
			},
			{
				Config: `
					resource scaleway_job_definition main {
						name = "test-jobs-job-definition-timeout"
						cpu_limit = 120
						memory_limit = 256
						image_uri = "docker.io/alpine:latest"
						timeout = "1h30m"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayJobDefinitionExists(tt, "scaleway_job_definition.main"),
					testCheckResourceAttrUUID("scaleway_job_definition.main", "id"),
					resource.TestCheckResourceAttr("scaleway_job_definition.main", "name", "test-jobs-job-definition-timeout"),
					resource.TestCheckResourceAttr("scaleway_job_definition.main", "timeout", "1h30m0s"),
				),
			},
		},
	})
}

func testAccCheckScalewayJobDefinitionExists(tt *TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := jobsAPIWithRegionAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetJobDefinition(&jobs.GetJobDefinitionRequest{
			JobDefinitionID: id,
			Region:          region,
		})

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayJobDefinitionDestroy(tt *TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_job_definition" {
				continue
			}

			api, region, id, err := jobsAPIWithRegionAndID(tt.Meta, rs.Primary.ID)
			if err != nil {
				return err
			}

			err = api.DeleteJobDefinition(&jobs.DeleteJobDefinitionRequest{
				JobDefinitionID: id,
				Region:          region,
			})

			if err == nil {
				return fmt.Errorf("jobs jobdefinition (%s) still exists", rs.Primary.ID)
			}

			if !is404Error(err) {
				return err
			}
		}

		return nil
	}
}
