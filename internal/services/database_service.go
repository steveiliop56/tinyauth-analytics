package services

import (
	"database/sql"
	"embed"

	"github.com/glebarez/sqlite"
	"github.com/golang-migrate/migrate/v4"
	sqliteMigrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

// Migrations
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

type DatabaseServiceConfig struct {
	DatabasePath string
}

type DatabaseService struct {
	config   DatabaseServiceConfig
	database *gorm.DB
}

func NewDatabaseService(config DatabaseServiceConfig) *DatabaseService {
	return &DatabaseService{
		config: config,
	}
}

func (ds *DatabaseService) Init() error {
	gormDB, err := gorm.Open(sqlite.Open(ds.config.DatabasePath), &gorm.Config{})

	if err != nil {
		return err
	}

	sqlDB, err := gormDB.DB()

	if err != nil {
		return err
	}

	sqlDB.SetMaxOpenConns(1)

	err = ds.migrateDatabase(sqlDB)

	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	ds.database = gormDB
	return nil
}

func (ds *DatabaseService) GetDatabase() *gorm.DB {
	return ds.database
}

func (ds *DatabaseService) migrateDatabase(sqlDB *sql.DB) error {
	data, err := iofs.New(migrationsFS, "migrations")

	if err != nil {
		return err
	}

	target, err := sqliteMigrate.WithInstance(sqlDB, &sqliteMigrate.Config{})

	if err != nil {
		return err
	}

	migrator, err := migrate.NewWithInstance("iofs", data, "tinyauth-analytics", target)

	if err != nil {
		return err
	}

	return migrator.Up()
}
