package postgres

import (
	"database/sql"
	"embed"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"

	mig "github.com/SversusN/gophermart/pkg/migrator"
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

func (p *Psql) Init(connectionString string) error {

	m := mig.MustGetNewMigrator(MigrationsFS, migrationsDir)
	err := m.ApplyMigrations(p.DB, connectionString)

	if err != nil {
		zap.Error(err)
		return err
	}

	return nil
}
