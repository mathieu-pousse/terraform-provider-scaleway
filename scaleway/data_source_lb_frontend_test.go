package scaleway

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceLbFrontend_Basic(t *testing.T) {
	tt := NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayLbIPDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_lb_ip ip01 {}
					resource scaleway_lb lb01 {
						ip_id = scaleway_lb_ip.ip01.id
						name = "test-lb"
						type = "lb-s"
					}
					resource scaleway_lb_backend bkd01 {
						lb_id = scaleway_lb.lb01.id
						forward_protocol = "tcp"
						forward_port = 80
						proxy_protocol = "none"
					}
					resource scaleway_lb_frontend frt01 {
						lb_id = scaleway_lb.lb01.id
						backend_id = scaleway_lb_backend.bkd01.id
						inbound_port = 80
					}
					
					data "scaleway_lb_frontend" "byID" {
						frontend_id = "${scaleway_lb_frontend.frt01.id}"
					}
					
					data "scaleway_lb_frontend" "byName" {
						name = "${scaleway_lb_frontend.frt01.name}"
						lb_id = "${scaleway_lb.lb01.id}"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.scaleway_lb_frontend.byID", "name",
						"scaleway_lb_frontend.frt01", "name"),
					resource.TestCheckResourceAttrPair(
						"data.scaleway_lb_frontend.byName", "id",
						"scaleway_lb_frontend.frt01", "id"),
				),
			},
		},
	})
}
