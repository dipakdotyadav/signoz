package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlschema"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type addProjectIngestionKeys struct { sqlschema sqlschema.SQLSchema }

func NewAddProjectIngestionKeysFactory(sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_project_ingestion_keys"), func(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
		return &addProjectIngestionKeys{sqlschema: sqlschema}, nil
	})
}

func (m *addProjectIngestionKeys) Register(migrations *migrate.Migrations) error {
	return migrations.Register(m.Up, m.Down)
}

func (m *addProjectIngestionKeys) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer func() { _ = tx.Rollback() }()

	tableSQL := m.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "project_ingestion_key",
		Columns: []*sqlschema.Column{
			{Name: "id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "org_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "project_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "name", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "key_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "secret_hash", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "expires_at", DataType: sqlschema.DataTypeTimestamp, Nullable: true},
			{Name: "revoked_at", DataType: sqlschema.DataTypeTimestamp, Nullable: true},
			{Name: "last_used_at", DataType: sqlschema.DataTypeTimestamp, Nullable: true},
			{Name: "created_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "updated_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"id"}},
		ForeignKeyConstraints: []*sqlschema.ForeignKeyConstraint{
			{ReferencingColumnName: "org_id", ReferencedTableName: "organizations", ReferencedColumnName: "id"},
			{ReferencingColumnName: "project_id", ReferencedTableName: "projects", ReferencedColumnName: "id"},
		},
	})
	indexSQL := []string{}
	for _, statement := range m.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "project_ingestion_key", ColumnNames: []sqlschema.ColumnName{"key_id"}}) {
		indexSQL = append(indexSQL, string(statement))
	}
	for _, statement := range m.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "project_ingestion_key", ColumnNames: []sqlschema.ColumnName{"org_id", "project_id", "name"}}) {
		indexSQL = append(indexSQL, string(statement))
	}
	for _, statement := range tableSQL {
		if _, err := tx.ExecContext(ctx, string(statement)); err != nil { return err }
	}
	for _, statement := range indexSQL {
		if _, err := tx.ExecContext(ctx, statement); err != nil { return err }
	}
	return tx.Commit()
}

func (m *addProjectIngestionKeys) Down(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS project_ingestion_key")
	return err
}
