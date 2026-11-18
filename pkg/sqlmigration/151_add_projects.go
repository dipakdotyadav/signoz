package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlschema"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type addProjects struct {
	sqlschema sqlschema.SQLSchema
}

func NewAddProjectsFactory(sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_projects"),
		func(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
			return &addProjects{sqlschema: sqlschema}, nil
		},
	)
}

func (migration *addProjects) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addProjects) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	tableSQL := migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "projects",
		Columns: []*sqlschema.Column{
			{Name: "id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "org_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "name", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "slug", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "description", DataType: sqlschema.DataTypeText, Nullable: true},
			{Name: "status", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "created_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "updated_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{
			ColumnNames: []sqlschema.ColumnName{"id"},
		},
		ForeignKeyConstraints: []*sqlschema.ForeignKeyConstraint{
			{
				ReferencingColumnName: sqlschema.ColumnName("org_id"),
				ReferencedTableName:   sqlschema.TableName("organizations"),
				ReferencedColumnName:  sqlschema.ColumnName("id"),
			},
		},
	})

	indexSQL := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{
		TableName:   sqlschema.TableName("projects"),
		ColumnNames: []sqlschema.ColumnName{"org_id", "slug"},
	})

	for _, statement := range append(tableSQL, indexSQL...) {
		if _, err := tx.ExecContext(ctx, string(statement)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (migration *addProjects) Down(context.Context, *bun.DB) error {
	return nil
}
