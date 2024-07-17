package postgres

import (
	"database/sql"
	"embed"
	"go.uber.org/zap"

	mig "github.com/SversusN/gophermart/pkg/migrator"

	_ "github.com/jackc/pgx/v4/stdlib"
)

type Psql struct {
	DB *sql.DB
}

//go:embed migrations/*.sql
var MigrationsFS embed.FS

const migrationsDir = "migrations"

func NewPsql(connectionString string) (*Psql, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, err
	}
	pgx := &Psql{DB: db}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return pgx, nil
}

func (p *Psql) Ping() error {
	if err := p.DB.Ping(); err != nil {
		return err
	}
	return nil
}

func (p *Psql) Init(dbName string) error {

	migrate := mig.MustGetNewMigrator(MigrationsFS, migrationsDir)
	err := migrate.ApplyMigrations(p.DB, dbName)

	if err != nil {
		zap.Error(err)
		return err
	}

	return nil
}
