package application

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-coolify/internal/api"
	"terraform-provider-coolify/internal/flatten"
)

type ApplicationResourceModel struct {
	Uuid            types.String `tfsdk:"uuid"`
	Name            types.String `tfsdk:"name"`
	ProjectUuid     types.String `tfsdk:"project_uuid"`
	ServerUuid      types.String `tfsdk:"server_uuid"`
	EnvironmentName types.String `tfsdk:"environment_name"`
	Description     types.String `tfsdk:"description"`

	// Source configuration
	SourceType              types.String `tfsdk:"source_type"` // docker_image, private_gh_app, public_source, etc.
	GitRepository           types.String `tfsdk:"git_repository"`
	GitBranch               types.String `tfsdk:"git_branch"`
	GitCommitSha            types.String `tfsdk:"git_commit_sha"`
	DockerRegistryImageName types.String `tfsdk:"docker_registry_image_name"`
	DockerRegistryImageTag  types.String `tfsdk:"docker_registry_image_tag"`
	Dockerfile              types.String `tfsdk:"dockerfile"`
	BuildPack               types.String `tfsdk:"build_pack"`
	PrivateKeyUuid          types.String `tfsdk:"private_key_uuid"`

	// Configuration
	BaseDirectory  types.String `tfsdk:"base_directory"`
	BuildCommand   types.String `tfsdk:"build_command"`
	StartCommand   types.String `tfsdk:"start_command"`
	InstallCommand types.String `tfsdk:"install_command"`
	PortsExposes   types.String `tfsdk:"ports_exposes"`
	PortsMappings  types.String `tfsdk:"ports_mappings"`

	// Advanced configuration
	DestinationUuid        types.String `tfsdk:"destination_uuid"`
	IsStatic               types.Bool   `tfsdk:"is_static"`
	InstantDeploy          types.Bool   `tfsdk:"instant_deploy"`
	WatchPaths             types.String `tfsdk:"watch_paths"`
	CustomDockerRunOptions types.String `tfsdk:"custom_docker_run_options"`
	CustomLabels           types.String `tfsdk:"custom_labels"`

	// Domains
	Domains types.String `tfsdk:"domains"`
}

func (m ApplicationResourceModel) Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages a Coolify Application.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:      true,
				Description:   "The unique identifier of the application.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the application.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A description of the application.",
			},
			"project_uuid": schema.StringAttribute{
				Required:      true,
				Description:   "The UUID of the project to deploy to.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"server_uuid": schema.StringAttribute{
				Required:      true,
				Description:   "The UUID of the server to deploy to.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_name": schema.StringAttribute{
				Required:      true,
				Description:   "The name of the environment (e.g., 'production').",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"source_type": schema.StringAttribute{
				Required:      true,
				Description:   "The type of application source: 'docker_image', 'public_source', 'private_gh_app', 'private_deploy_key', 'dockerfile'.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"git_repository": schema.StringAttribute{
				Optional:    true,
				Description: "The Git repository URL (required for git-based sources).",
			},
			"git_branch": schema.StringAttribute{
				Optional:    true,
				Description: "The Git branch to deploy from.",
				Default:     stringdefault.StaticString("main"),
				Computed:    true,
			},
			"git_commit_sha": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The specific Git commit SHA to deploy.",
			},
			"docker_registry_image_name": schema.StringAttribute{
				Optional:    true,
				Description: "The Docker image name (required for 'docker_image').",
			},
			"docker_registry_image_tag": schema.StringAttribute{
				Optional:    true,
				Description: "The Docker image tag.",
				Default:     stringdefault.StaticString("latest"),
				Computed:    true,
			},
			"dockerfile": schema.StringAttribute{
				Optional:    true,
				Description: "The content of the Dockerfile (required for 'dockerfile' type).",
			},
			"build_pack": schema.StringAttribute{
				Optional:    true,
				Description: "The build pack to use (nixpacks, static, dockerfile, dockercompose).",
				Default:     stringdefault.StaticString("nixpacks"),
				Computed:    true,
			},
			"private_key_uuid": schema.StringAttribute{
				Optional:    true,
				Description: "The UUID of the private key for private repositories.",
			},
			"base_directory": schema.StringAttribute{
				Optional:    true,
				Description: "The base directory for the build.",
			},
			"build_command": schema.StringAttribute{
				Optional:    true,
				Description: "The command to build the application.",
			},
			"start_command": schema.StringAttribute{
				Optional:    true,
				Description: "The command to start the application.",
			},
			"install_command": schema.StringAttribute{
				Optional:    true,
				Description: "The command to install dependencies.",
			},
			"ports_exposes": schema.StringAttribute{
				Optional:    true,
				Description: "Comma separated list of ports exposed by the container.",
			},
			"ports_mappings": schema.StringAttribute{
				Optional:    true,
				Description: "Comma separated list of port mappings (host:container).",
			},
			"destination_uuid": schema.StringAttribute{
				Computed:      true,
				Optional:      true,
				Description:   "The UUID of the destination (network network).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"is_static": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the application is a static site.",
				Default:     booldefault.StaticBool(false),
			},
			"instant_deploy": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to deploy instantly upon creation.",
				Default:     booldefault.StaticBool(false),
			},
			"watch_paths": schema.StringAttribute{
				Optional:    true,
				Description: "Paths to watch for changes.",
			},
			"custom_docker_run_options": schema.StringAttribute{
				Optional:    true,
				Description: "Custom options to pass to 'docker run'.",
			},
			"custom_labels": schema.StringAttribute{
				Optional:    true,
				Description: "Custom labels for the container.",
			},
			"domains": schema.StringAttribute{
				Optional:    true,
				Description: "The domains configurations for the application.",
			},
		},
	}
}

func (m ApplicationResourceModel) FromAPI(data *api.Application) ApplicationResourceModel {
	// Helper to merge state with API data
	model := ApplicationResourceModel{
		Uuid:            flatten.String(data.Uuid),
		Name:            flatten.String(data.Name),
		Description:     flatten.String(data.Description),
		EnvironmentName: types.StringValue(""), // API doesn't always return this, stored in state

		GitRepository:           flatten.String(data.GitRepository),
		GitBranch:               flatten.String(data.GitBranch),
		GitCommitSha:            flatten.String(data.GitCommitSha),
		DockerRegistryImageName: flatten.String(data.DockerRegistryImageName),
		DockerRegistryImageTag:  flatten.String(data.DockerRegistryImageTag),
		Dockerfile:              flatten.String(data.Dockerfile),
		BuildPack:               flatten.String((*string)(data.BuildPack)),

		BaseDirectory:  flatten.String(data.BaseDirectory),
		BuildCommand:   flatten.String(data.BuildCommand),
		StartCommand:   flatten.String(data.StartCommand),
		InstallCommand: flatten.String(data.InstallCommand),
		PortsExposes:   flatten.String(data.PortsExposes),
		PortsMappings:  flatten.String(data.PortsMappings),

		WatchPaths:             flatten.String(data.WatchPaths),
		CustomDockerRunOptions: flatten.String(data.CustomDockerRunOptions),
		CustomLabels:           flatten.String(data.CustomLabels),
		Domains:                flatten.String(data.Fqdn),
	}
	return model
}
