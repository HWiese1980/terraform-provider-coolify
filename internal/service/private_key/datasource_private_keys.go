package private_key

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"terraform-provider-coolify/internal/api"
	"terraform-provider-coolify/internal/filter"
	"terraform-provider-coolify/internal/provider/util"
)

var _ datasource.DataSource = &privateKeysDataSource{}
var _ datasource.DataSourceWithConfigure = &privateKeysDataSource{}

func NewPrivateKeysDataSource() datasource.DataSource {
	return &privateKeysDataSource{}
}

type privateKeysDataSource struct {
	client *api.ClientWithResponses
}

func (d *privateKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_keys"
}

func (d *privateKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {

	ds := NewPrivateKeyDataSource()
	dsResp := datasource.SchemaResponse{}

	ds.Schema(ctx, datasource.SchemaRequest{}, &dsResp)

	if attr, ok := dsResp.Schema.Attributes["uuid"].(schema.StringAttribute); ok {
		attr.Required = false
		attr.Computed = true
		dsResp.Schema.Attributes["uuid"] = attr
	}

	resp.Schema = schema.Schema{
		Description:         "Get a list of Coolify private keys.",
		MarkdownDescription: "Get a list of Coolify private keys.",
		Attributes: map[string]schema.Attribute{
			"private_keys": schema.SetNestedAttribute{
				Description: "List of private keys",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: dsResp.Schema.Attributes,
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.CreateDatasourceFilter(privateKeysFilterNames),
		},
	}
}

func (d *privateKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	util.ProviderDataFromDataSourceConfigureRequest(req, &d.client, resp)
}

func (d *privateKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var plan privateKeysDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	listResponse, err := d.client.ListPrivateKeysWithResponse(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading private keys", err.Error(),
		)
		return
	}

	if listResponse.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected HTTP status code reading private key",
			fmt.Sprintf("Received %s for private keys. Details: %s", listResponse.Status(), listResponse.Body),
		)
		return
	}

	state, diag := d.apiToModel(ctx, listResponse.JSON200, plan.Filter)
	resp.Diagnostics.Append(diag...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *privateKeysDataSource) apiToModel(
	ctx context.Context,
	privateKeys *[]api.PrivateKey,
	filters []filter.BlockModel,
) (privateKeysDataSourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var privateKeyValues []privateKeyDataSourceModel

	for _, pk := range *privateKeys {
		model := privateKeyDataSourceModel{}.FromAPI(&pk)

		if !filter.OnStruct(ctx, model, filters) {
			continue
		}

		privateKeyValues = append(privateKeyValues, model)
	}

	return privateKeysDataSourceModel{
		PrivateKeys: privateKeyValues,
		Filter:      filters,
	}, diags
}
