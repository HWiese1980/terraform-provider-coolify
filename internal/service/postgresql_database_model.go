package service

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-coolify/internal/api"
	"terraform-provider-coolify/internal/flatten"
)

type postgresqlDatabaseModel struct {
	commonDatabaseModel
	PostgresConf           types.String `tfsdk:"postgres_conf"`
	PostgresDb             types.String `tfsdk:"postgres_db"`
	PostgresHostAuthMethod types.String `tfsdk:"postgres_host_auth_method"`
	PostgresInitdbArgs     types.String `tfsdk:"postgres_initdb_args"`
	PostgresPassword       types.String `tfsdk:"postgres_password"`
	PostgresUser           types.String `tfsdk:"postgres_user"`
}

func (m postgresqlDatabaseModel) FromAPI(apiModel *api.Database, state postgresqlDatabaseModel) (postgresqlDatabaseModel, error) {
	apiModel.ValueByDiscriminator()
	db, err := apiModel.AsPostgresqlDatabase()
	if err != nil {
		return postgresqlDatabaseModel{}, err
	}

	return postgresqlDatabaseModel{
		commonDatabaseModel:    commonDatabaseModel{}.FromAPI(apiModel, state.commonDatabaseModel),
		PostgresConf:           flatten.String(db.PostgresConf),
		PostgresDb:             flatten.String(db.PostgresDb),
		PostgresHostAuthMethod: flatten.String(db.PostgresHostAuthMethod),
		PostgresInitdbArgs:     flatten.String(db.PostgresInitdbArgs),
		PostgresPassword:       flatten.String(db.PostgresPassword),
		PostgresUser:           flatten.String(db.PostgresUser),
	}, nil
}
