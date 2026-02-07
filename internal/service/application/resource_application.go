package application

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-coolify/internal/api"
	"terraform-provider-coolify/internal/provider/util"
)

var (
	_ resource.Resource                = &ApplicationResource{}
	_ resource.ResourceWithConfigure   = &ApplicationResource{}
	_ resource.ResourceWithImportState = &ApplicationResource{}
)

func NewApplicationResource() resource.Resource {
	return &ApplicationResource{}
}

type ApplicationResource struct {
	client *api.ClientWithResponses
}

func (r *ApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *ApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ApplicationResourceModel{}.Schema(ctx)
}

func (r *ApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	util.ProviderDataFromResourceConfigureRequest(req, &r.client, resp)
}

func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceType := plan.SourceType.ValueString()
	projectUuid := plan.ProjectUuid.ValueString()
	serverUuid := plan.ServerUuid.ValueString()
	environmentName := plan.EnvironmentName.ValueString()
	name := plan.Name.ValueString()

	tflog.Debug(ctx, "Creating application", map[string]interface{}{
		"name":        name,
		"source_type": sourceType,
	})

	var uuid *string

	switch sourceType {
	case "docker_image":
		image := plan.DockerRegistryImageName.ValueString()
		tag := plan.DockerRegistryImageTag.ValueString()
		res, err := r.client.CreateDockerimageApplicationWithResponse(ctx, api.CreateDockerimageApplicationJSONRequestBody{
			ProjectUuid:             projectUuid,
			ServerUuid:              serverUuid,
			EnvironmentName:         environmentName,
			Name:                    &name,
			DockerRegistryImageName: image,
			DockerRegistryImageTag:  &tag,
			Description:             plan.Description.ValueStringPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating docker image application", err.Error())
			return
		}
		if res.StatusCode() != http.StatusCreated {
			resp.Diagnostics.AddError("Unexpected status code creating docker image application", fmt.Sprintf("%s: %s", res.Status(), res.Body))
			return
		}
		uuid = res.JSON201.Uuid

	case "public_source":
		repo := plan.GitRepository.ValueString()
		branch := plan.GitBranch.ValueString()
		res, err := r.client.CreatePublicApplicationWithResponse(ctx, api.CreatePublicApplicationJSONRequestBody{
			ProjectUuid:     projectUuid,
			ServerUuid:      serverUuid,
			EnvironmentName: environmentName,
			Name:            &name,
			GitRepository:   repo,
			GitBranch:       branch,
			Description:     plan.Description.ValueStringPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating public source application", err.Error())
			return
		}
		if res.StatusCode() != http.StatusCreated {
			resp.Diagnostics.AddError("Unexpected status code creating public source application", fmt.Sprintf("%s: %s", res.Status(), res.Body))
			return
		}
		uuid = res.JSON201.Uuid

	case "private_deploy_key":
		repo := plan.GitRepository.ValueString()
		branch := plan.GitBranch.ValueString()
		privateKeyUuid := plan.PrivateKeyUuid.ValueString()
		res, err := r.client.CreatePrivateDeployKeyApplicationWithResponse(ctx, api.CreatePrivateDeployKeyApplicationJSONRequestBody{
			ProjectUuid:     projectUuid,
			ServerUuid:      serverUuid,
			EnvironmentName: environmentName,
			Name:            &name,
			GitRepository:   repo,
			GitBranch:       branch,
			PrivateKeyUuid:  privateKeyUuid,
			Description:     plan.Description.ValueStringPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating private deploy key application", err.Error())
			return
		}
		if res.StatusCode() != http.StatusCreated {
			resp.Diagnostics.AddError("Unexpected status code creating private deploy key application", fmt.Sprintf("%s: %s", res.Status(), res.Body))
			return
		}
		uuid = res.JSON201.Uuid

	case "dockerfile":
		content := plan.Dockerfile.ValueString()
		res, err := r.client.CreateDockerfileApplicationWithResponse(ctx, api.CreateDockerfileApplicationJSONRequestBody{
			ProjectUuid:     projectUuid,
			ServerUuid:      serverUuid,
			EnvironmentName: environmentName,
			Name:            &name,
			Dockerfile:      content,
			Description:     plan.Description.ValueStringPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating dockerfile application", err.Error())
			return
		}
		if res.StatusCode() != http.StatusCreated {
			resp.Diagnostics.AddError("Unexpected status code creating dockerfile application", fmt.Sprintf("%s: %s", res.Status(), res.Body))
			return
		}
		uuid = res.JSON201.Uuid

	default:
		resp.Diagnostics.AddError("Invalid source_type", fmt.Sprintf("Source type '%s' is not supported. Supported: docker_image, public_source, private_deploy_key, dockerfile", sourceType))
		return
	}

	if uuid == nil {
		resp.Diagnostics.AddError("Error creating application", "API returned no UUID")
		return
	}

	// Update the rest of the configuration
	updateBody := api.UpdateApplicationByUuidJSONRequestBody{
		Description:            plan.Description.ValueStringPointer(),
		BuildCommand:           plan.BuildCommand.ValueStringPointer(),
		StartCommand:           plan.StartCommand.ValueStringPointer(),
		InstallCommand:         plan.InstallCommand.ValueStringPointer(),
		PortsExposes:           plan.PortsExposes.ValueStringPointer(),
		PortsMappings:          plan.PortsMappings.ValueStringPointer(),
		BaseDirectory:          plan.BaseDirectory.ValueStringPointer(),
		IsStatic:               plan.IsStatic.ValueBoolPointer(),
		WatchPaths:             plan.WatchPaths.ValueStringPointer(),
		CustomDockerRunOptions: plan.CustomDockerRunOptions.ValueStringPointer(),
		CustomLabels:           plan.CustomLabels.ValueStringPointer(),
		Domains:                plan.Domains.ValueStringPointer(),
	}

	// Only set enum fields if they are set (to avoid overwriting defaults or causing issues)
	// BuildPack is an enum
	if !plan.BuildPack.IsNull() && !plan.BuildPack.IsUnknown() {
		bp := api.UpdateApplicationByUuidJSONBodyBuildPack(plan.BuildPack.ValueString())
		updateBody.BuildPack = &bp
	}

	tflog.Debug(ctx, "Updating application configuration", map[string]interface{}{"uuid": *uuid})
	updateRes, err := r.client.UpdateApplicationByUuidWithResponse(ctx, *uuid, updateBody)
	if err != nil {
		resp.Diagnostics.AddError("Error updating application config", err.Error())
		// Don't return, try to read state at least? No, incomplete state is bad.
		return
	}
	if updateRes.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Unexpected status code updating application config", fmt.Sprintf("%s: %s", updateRes.Status(), updateRes.Body))
		return
	}

	// Deploy if requested
	if plan.InstantDeploy.ValueBool() {
		deployRes, err := r.client.RestartApplicationByUuidWithResponse(ctx, *uuid)
		if err != nil {
			resp.Diagnostics.AddWarning("Error triggering deploy", err.Error())
		} else if deployRes.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddWarning("Unexpected status code triggering deploy", fmt.Sprintf("%s: %s", deployRes.Status(), deployRes.Body))
		}
	}

	// Read back state
	data, ok := r.ReadFromAPI(ctx, &resp.Diagnostics, *uuid, plan)
	if !ok {
		return // ReadFromAPI already added error
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, ok := r.ReadFromAPI(ctx, &resp.Diagnostics, state.Uuid.ValueString(), state)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := plan.Uuid.ValueString()

	updateBody := api.UpdateApplicationByUuidJSONRequestBody{
		Name:                   plan.Name.ValueStringPointer(),
		Description:            plan.Description.ValueStringPointer(),
		BuildCommand:           plan.BuildCommand.ValueStringPointer(),
		StartCommand:           plan.StartCommand.ValueStringPointer(),
		InstallCommand:         plan.InstallCommand.ValueStringPointer(),
		PortsExposes:           plan.PortsExposes.ValueStringPointer(),
		PortsMappings:          plan.PortsMappings.ValueStringPointer(),
		BaseDirectory:          plan.BaseDirectory.ValueStringPointer(),
		IsStatic:               plan.IsStatic.ValueBoolPointer(),
		WatchPaths:             plan.WatchPaths.ValueStringPointer(),
		CustomDockerRunOptions: plan.CustomDockerRunOptions.ValueStringPointer(),
		CustomLabels:           plan.CustomLabels.ValueStringPointer(),
		Domains:                plan.Domains.ValueStringPointer(),
		GitBranch:              plan.GitBranch.ValueStringPointer(),
		GitCommitSha:           plan.GitCommitSha.ValueStringPointer(),
		DockerRegistryImageTag: plan.DockerRegistryImageTag.ValueStringPointer(),
		Dockerfile:             plan.Dockerfile.ValueStringPointer(),
	}

	if !plan.BuildPack.IsNull() && !plan.BuildPack.IsUnknown() {
		bp := api.UpdateApplicationByUuidJSONBodyBuildPack(plan.BuildPack.ValueString())
		updateBody.BuildPack = &bp
	}

	res, err := r.client.UpdateApplicationByUuidWithResponse(ctx, uuid, updateBody)
	if err != nil {
		resp.Diagnostics.AddError("Error updating application", err.Error())
		return
	}
	if res.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Unexpected status code updating application", fmt.Sprintf("%s: %s", res.Status(), res.Body))
		return
	}

	if plan.InstantDeploy.ValueBool() {
		deployRes, err := r.client.RestartApplicationByUuidWithResponse(ctx, uuid)
		if err != nil {
			resp.Diagnostics.AddWarning("Error triggering deploy", err.Error())
		} else if deployRes.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddWarning("Unexpected status code triggering deploy", fmt.Sprintf("%s: %s", deployRes.Status(), deployRes.Body))
		}
	}

	data, ok := r.ReadFromAPI(ctx, &resp.Diagnostics, uuid, plan)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.Uuid.ValueString()
	res, err := r.client.DeleteApplicationByUuidWithResponse(ctx, uuid, &api.DeleteApplicationByUuidParams{
		DeleteConfigurations: types.BoolValue(true).ValueBoolPointer(),
		DeleteVolumes:        types.BoolValue(true).ValueBoolPointer(),
		DockerCleanup:        types.BoolValue(true).ValueBoolPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting application", err.Error())
		return
	}
	if res.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Unexpected status code deleting application", fmt.Sprintf("%s: %s", res.Status(), res.Body))
		return
	}
}

func (r *ApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func (r *ApplicationResource) ReadFromAPI(ctx context.Context, diags *diag.Diagnostics, uuid string, state ApplicationResourceModel) (ApplicationResourceModel, bool) {
	res, err := r.client.GetApplicationByUuidWithResponse(ctx, uuid)
	if err != nil {
		diags.AddError("Error reading application", err.Error())
		return ApplicationResourceModel{}, false
	}
	if res.StatusCode() == http.StatusNotFound {
		return ApplicationResourceModel{}, false
	}
	if res.StatusCode() != http.StatusOK {
		diags.AddError("Unexpected status code reading application", fmt.Sprintf("%s: %s", res.Status(), res.Body))
		return ApplicationResourceModel{}, false
	}

	// Helper to merge state with API data
	model := ApplicationResourceModel{}.FromAPI(res.JSON200)

	// Preserve fields not returned by API or slightly different
	model.SourceType = state.SourceType
	model.ProjectUuid = state.ProjectUuid
	model.ServerUuid = state.ServerUuid
	model.EnvironmentName = state.EnvironmentName
	model.PrivateKeyUuid = state.PrivateKeyUuid
	model.InstantDeploy = state.InstantDeploy

	// Some fields might be empty in API but set in state (secrets etc, though we marked those sensitive in datasource not here yet)
	// For now, simple merge

	return model, true
}
