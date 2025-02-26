package scaleway

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	accountV3 "github.com/scaleway/scaleway-sdk-go/api/account/v3"
	mnq "github.com/scaleway/scaleway-sdk-go/api/mnq/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccScalewayMNQSQSQueue_Basic(t *testing.T) {
	tt := NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayMNQSQSQueueDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_account_project main {
						name = "tf_tests_mnq_sqs_queue_basic"
					}

					resource scaleway_mnq_sqs main {
						project_id = scaleway_account_project.main.id
					}

					resource scaleway_mnq_sqs_credentials main {
						project_id = scaleway_mnq_sqs.main.project_id
						permissions {
							can_manage = true
						}
					}

					resource scaleway_mnq_sqs_queue main {
						project_id = scaleway_mnq_sqs.main.project_id
						name = "test-mnq-sqs-queue-basic"
						sqs_endpoint = scaleway_mnq_sqs.main.endpoint
						access_key = scaleway_mnq_sqs_credentials.main.access_key
						secret_key = scaleway_mnq_sqs_credentials.main.secret_key
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayMNQSQSQueueExists(tt, "scaleway_mnq_sqs_queue.main"),
					testCheckResourceAttrUUID("scaleway_mnq_sqs_queue.main", "id"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_queue.main", "name", "test-mnq-sqs-queue-basic"),
				),
			},
			{
				Config: `
					resource scaleway_account_project main {
						name = "tf_tests_mnq_sqs_queue_basic"
					}

					resource scaleway_mnq_sqs main {
						project_id = scaleway_account_project.main.id
					}

					resource scaleway_mnq_sqs_credentials main {
						project_id = scaleway_mnq_sqs.main.project_id
						permissions {
							can_manage = true
						}
					}

					resource scaleway_mnq_sqs_queue main {
						project_id = scaleway_mnq_sqs.main.project_id
						name = "test-mnq-sqs-queue-basic"
						sqs_endpoint = scaleway_mnq_sqs.main.endpoint
						access_key = scaleway_mnq_sqs_credentials.main.access_key
						secret_key = scaleway_mnq_sqs_credentials.main.secret_key

						message_max_age = 720
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayMNQSQSQueueExists(tt, "scaleway_mnq_sqs_queue.main"),
					testCheckResourceAttrUUID("scaleway_mnq_sqs_queue.main", "id"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_queue.main", "message_max_age", "720"),
				),
			},
		},
	})
}

func TestAccScalewayMNQSQSQueue_DefaultProject(t *testing.T) {
	tt := NewTestTools(t)
	defer tt.Cleanup()

	ctx := context.Background()

	accountAPI := accountV3.NewProjectAPI(tt.Meta.scwClient)
	projectID := ""
	project, err := accountAPI.CreateProject(&accountV3.ProjectAPICreateProjectRequest{
		Name: "tf_tests_mnq_sqs_queue_default_project",
	})
	assert.Nil(t, err)

	projectID = project.ID

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: func() map[string]func() (*schema.Provider, error) {
			metaProd, err := buildMeta(ctx, &metaConfig{
				terraformVersion: "terraform-tests",
				httpClient:       tt.Meta.httpClient,
			})
			require.NoError(t, err)

			return map[string]func() (*schema.Provider, error){
				"scaleway": func() (*schema.Provider, error) {
					return Provider(&ProviderConfig{Meta: metaProd})(), nil
				},
			}
		}(),
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayMNQSQSQueueDestroy(tt),
			func(state *terraform.State) error {
				return accountAPI.DeleteProject(&accountV3.ProjectAPIDeleteProjectRequest{
					ProjectID: projectID,
				})
			},
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_mnq_sqs main {
						project_id = "%1s"
					}

					resource scaleway_mnq_sqs_credentials main {
						project_id = scaleway_mnq_sqs.main.project_id
						permissions {
							can_manage = true
						}
					}

					resource scaleway_mnq_sqs_queue main {
						project_id = scaleway_mnq_sqs.main.project_id
						name = "test-mnq-sqs-queue-basic"
						access_key = scaleway_mnq_sqs_credentials.main.access_key
						secret_key = scaleway_mnq_sqs_credentials.main.secret_key
					}
				`, projectID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayMNQSQSQueueExists(tt, "scaleway_mnq_sqs_queue.main"),
					testCheckResourceAttrUUID("scaleway_mnq_sqs_queue.main", "id"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_queue.main", "name", "test-mnq-sqs-queue-basic"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_queue.main", "project_id", projectID),
				),
			},
		},
	})
}

func testAccCheckScalewayMNQSQSQueueExists(tt *TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		region, _, queueName, err := decomposeMNQID(rs.Primary.ID)
		if err != nil {
			return err
		}

		sqsClient, err := newSQSClient(tt.Meta.httpClient, region.String(), rs.Primary.Attributes["sqs_endpoint"], rs.Primary.Attributes["access_key"], rs.Primary.Attributes["secret_key"])
		if err != nil {
			return err
		}

		_, err = sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
			QueueName: aws.String(queueName),
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayMNQSQSQueueDestroy(tt *TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_mnq_sqs_queue" {
				continue
			}

			region, projectID, queueName, err := decomposeMNQID(rs.Primary.ID)
			if err != nil {
				return err
			}

			// Project may have been deleted, check for it first
			// Checking for Queue first may lead to an AccessDenied if project has been deleted
			accountAPI := accountV3ProjectAPI(tt.Meta)
			_, err = accountAPI.GetProject(&accountV3.ProjectAPIGetProjectRequest{
				ProjectID: projectID,
			})
			if err != nil {
				if is404Error(err) {
					return nil
				}

				return err
			}

			mnqAPI := mnq.NewSqsAPI(tt.Meta.scwClient)
			sqsInfo, err := mnqAPI.GetSqsInfo(&mnq.SqsAPIGetSqsInfoRequest{
				Region:    region,
				ProjectID: projectID,
			})
			if err != nil {
				return err
			}

			// SQS may be disabled for project, this means the queue does not exist
			if sqsInfo.Status == mnq.SqsInfoStatusDisabled {
				return nil
			}

			sqsClient, err := newSQSClient(tt.Meta.httpClient, region.String(), rs.Primary.Attributes["sqs_endpoint"], rs.Primary.Attributes["access_key"], rs.Primary.Attributes["secret_key"])
			if err != nil {
				return err
			}

			_, err = sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
				QueueName: aws.String(queueName),
			})
			if err != nil {
				if tfawserr.ErrCodeEquals(err, sqs.ErrCodeQueueDoesNotExist) || tfawserr.ErrCodeEquals(err, "AccessDeniedException") {
					return nil
				}

				return fmt.Errorf("failed to get queue url: %s", err)
			}

			if err == nil {
				return fmt.Errorf("mnq sqs queue (%s) still exists", rs.Primary.ID)
			}

			if !is404Error(err) {
				return err
			}
		}

		return nil
	}
}
