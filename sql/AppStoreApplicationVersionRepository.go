package sql

import (
	"github.com/go-pg/pg"
	"time"
)

type AppStoreApplicationVersionRepository interface {
	FindAll() ([]AppStoreApplicationVersion, error)
	GetInstalledAppVersionByAppStoreApplicationVersionId(appStoreApplicationVersionId int) ([]*InstalledAppVersions, error)
	DeleteAppStoreApplicationVersion(appStoreVersionId int) error
	UpdateInstalledAppVersion(model *InstalledAppVersions, tx *pg.Tx) (*InstalledAppVersions, error)
	UpdateAppStoreApplicationVersion(appStoreApplicationVersionId int) error
}

type AppStoreApplicationVersionRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewAppStoreApplicationVersionRepositoryImpl(dbConnection *pg.DB) *AppStoreApplicationVersionRepositoryImpl {
	return &AppStoreApplicationVersionRepositoryImpl{dbConnection: dbConnection}
}

type AppStoreApplicationVersion struct {
	TableName   struct{}  `sql:"app_store_application_version" pg:",discard_unknown_columns"`
	Id          int       `sql:"id,pk"`
	Version     string    `sql:"version"`
	AppVersion  string    `sql:"app_version"`
	Created     time.Time `sql:"created"`
	Deprecated  bool      `sql:"deprecated,notnull"`
	Description string    `sql:"description"`
	Digest      string    `sql:"digest"`
	Icon        string    `sql:"icon"`
	Name        string    `sql:"name"`
	Source      string    `sql:"source"`
	Home        string    `sql:"home"`
	ValuesYaml  string    `sql:"values_yaml"`
	ChartYaml   string    `sql:"chart_yaml"`
	Latest      bool      `sql:"latest,notnull"`
	AppStoreId  int       `sql:"app_store_id"`
	AuditLog
	RawValues        string `sql:"raw_values"`
	Readme           string `sql:"readme"`
	ValuesSchemaJson string `sql:"values_schema_json"`
	Notes            string `sql:"notes"`
	AppStore         *AppStore
}

type AuditLog struct {
	CreatedOn time.Time `sql:"created_on"`
	CreatedBy int32     `sql:"created_by"`
	UpdatedOn time.Time `sql:"updated_on"`
	UpdatedBy int32     `sql:"updated_by"`
}

type InstalledAppVersions struct {
	TableName                    struct{} `sql:"installed_app_versions" pg:",discard_unknown_columns"`
	Id                           int      `sql:"id,pk"`
	InstalledAppId               int      `sql:"installed_app_id,notnull"`
	AppStoreApplicationVersionId int      `sql:"app_store_application_version_id,notnull"`
	ValuesYaml                   string   `sql:"values_yaml_raw"`
	Active                       bool     `sql:"active, notnull"`
	ReferenceValueId             int      `sql:"reference_value_id"`
	ReferenceValueKind           string   `sql:"reference_value_kind"`
	AuditLog
	AppStoreApplicationVersion AppStoreApplicationVersion
}

type AppStore struct {
	TableName             struct{}  `sql:"app_store" pg:",discard_unknown_columns"`
	Id                    int       `sql:"id,pk"`
	Name                  string    `sql:"name,notnull"`
	ChartRepoId           int       `sql:"chart_repo_id"`
	DockerArtifactStoreId string    `sql:"docker_artifact_store_id"`
	Active                bool      `sql:"active,notnull"`
	ChartGitLocation      string    `sql:"chart_git_location"`
	CreatedOn             time.Time `sql:"created_on,notnull"`
	UpdatedOn             time.Time `sql:"updated_on,notnull"`
}

func (impl AppStoreApplicationVersionRepositoryImpl) FindAll() ([]AppStoreApplicationVersion, error) {
	var appStoreWithVersion []AppStoreApplicationVersion
	queryTemp := "SELECT t.id,t.name,t.version, t.app_store_id,t.latest FROM ( SELECT s.*, COUNT(*) OVER (PARTITION BY s.app_store_id,s.name, s.version) AS qty FROM app_store_application_version s )  t where (t.qty>1);"
	_, err := impl.dbConnection.Query(&appStoreWithVersion, queryTemp, true)
	if err != nil {
		return nil, err
	}
	return appStoreWithVersion, err
}

func (impl AppStoreApplicationVersionRepositoryImpl) GetInstalledAppVersionByAppStoreApplicationVersionId(appStoreApplicationVersionId int) ([]*InstalledAppVersions, error) {
	var model []*InstalledAppVersions
	err := impl.dbConnection.Model(&model).
		Column("installed_app_versions.*", "AppStoreApplicationVersion").
		Where("app_store_application_version.id = ?", appStoreApplicationVersionId).Select()
	if err != nil {
		return model, err
	}
	return model, nil
}

func (impl AppStoreApplicationVersionRepositoryImpl) DeleteAppStoreApplicationVersion(appStoreVersionId int) error {
	appStoreApplicationVersionDeleteQuery := "delete from app_store_application_version where id = ?"
	_, err := impl.dbConnection.Exec(appStoreApplicationVersionDeleteQuery, appStoreVersionId)
	if err != nil {
		return err
	}
	return nil
}

func (impl AppStoreApplicationVersionRepositoryImpl) UpdateInstalledAppVersion(model *InstalledAppVersions, tx *pg.Tx) (*InstalledAppVersions, error) {
	var err error
	if tx == nil {
		err = impl.dbConnection.Update(model)
	} else {
		err = tx.Update(model)
	}
	if err != nil {
		return model, err
	}
	return model, nil
}

func (impl AppStoreApplicationVersionRepositoryImpl) UpdateAppStoreApplicationVersion(appStoreApplicationVersionId int) error {
	_, err := impl.dbConnection.Exec("update app_store_application_version set latest=true where id=?", appStoreApplicationVersionId)
	if err != nil {
		return err
	}
	return nil
}
