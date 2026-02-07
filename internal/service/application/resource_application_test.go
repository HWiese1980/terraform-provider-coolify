package application_test

import (
	"context"
	"testing"

	"terraform-provider-coolify/internal/acctest"
	"terraform-provider-coolify/internal/service/application"

	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApplicationResource(t *testing.T) {
	resName := "coolify_application.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // Create and Read testing
				Config: `
				resource "coolify_project" "test" {
					name = "TerraformAppProject"
				}

				resource "coolify_server" "test" {
					name = "TerraformAppServer"
					ip   = "localhost"
					port = 22
					user = "root"
					private_key_uuid = "` + acctest.PrivateKeyUUID + `"
					instant_validate = false
				}

				resource "coolify_application" "test" {
					name                       = "terraform-app-test"
					project_uuid               = coolify_project.test.uuid
					server_uuid                = coolify_server.test.uuid
					environment_name           = "production"
					source_type                = "docker_image"
					docker_registry_image_name = "nginx"
					docker_registry_image_tag  = "latest"
					ports_exposes              = "80"
					instant_deploy             = false 
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resName, "name", "terraform-app-test"),
					resource.TestCheckResourceAttr(resName, "source_type", "docker_image"),
					resource.TestCheckResourceAttr(resName, "docker_registry_image_name", "nginx"),
					resource.TestCheckResourceAttr(resName, "docker_registry_image_tag", "latest"),
					resource.TestCheckResourceAttrSet(resName, "uuid"),
				),
			},
			{ // Update and Read testing
				Config: `
				resource "coolify_project" "test" {
					name = "TerraformAppProject"
				}

				resource "coolify_server" "test" {
					name = "TerraformAppServer"
					ip   = "localhost"
					port = 22
					user = "root"
					private_key_uuid = "` + acctest.PrivateKeyUUID + `"
					instant_validate = false
				}

				resource "coolify_application" "test" {
					name                       = "terraform-app-test-updated"
					project_uuid               = coolify_project.test.uuid
					server_uuid                = coolify_server.test.uuid
					environment_name           = "production" // Force new if changed, so keep same
					source_type                = "docker_image"
					docker_registry_image_name = "nginx"
					docker_registry_image_tag  = "alpine"
					ports_exposes              = "8080"
					instant_deploy             = false
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resName, "name", "terraform-app-test-updated"),
					resource.TestCheckResourceAttr(resName, "docker_registry_image_tag", "alpine"),
					resource.TestCheckResourceAttr(resName, "ports_exposes", "8080"),
				),
			},
		},
	})
}

func TestApplicationResourceSchema(t *testing.T) {
	ctx := context.Background()
	rs := application.NewApplicationResource()
	resp := &tfresource.SchemaResponse{}
	rs.Schema(ctx, tfresource.SchemaRequest{}, resp)
}
