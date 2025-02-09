package mysql

import (
	"fmt"

	"github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/contracts/database"
	"github.com/goravel/framework/contracts/database/driver"
	contractsschema "github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/contracts/log"
	"github.com/goravel/framework/contracts/testing/docker"
	"github.com/goravel/framework/errors"
	"github.com/goravel/framework/support/str"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"

	"github.com/goravel/mysql/contracts"
)

var _ driver.Driver = &Mysql{}

type Mysql struct {
	config contracts.ConfigBuilder
	db     *gorm.DB
	log    log.Log
}

func NewMysql(config config.Config, log log.Log, connection string) *Mysql {
	return &Mysql{
		config: NewConfig(config, connection),
		log:    log,
	}
}

func (r *Mysql) Config() database.Config {
	writers := r.config.Writes()
	if len(writers) == 0 {
		return database.Config{}
	}

	version, name := r.versionAndName()

	return database.Config{
		Connection: writers[0].Connection,
		Database:   writers[0].Database,
		Driver:     name,
		Host:       writers[0].Host,
		Password:   writers[0].Password,
		Port:       writers[0].Port,
		Prefix:     writers[0].Prefix,
		Username:   writers[0].Username,
		Version:    version,
	}
}

func (r *Mysql) DB() (*sqlx.DB, error) {
	gormDB, _, err := r.Gorm()
	if err != nil {
		return nil, err
	}

	db, err := gormDB.DB()
	if err != nil {
		return nil, err
	}

	return sqlx.NewDb(db, Name), nil
}

func (r *Mysql) Docker() (docker.DatabaseDriver, error) {
	writers := r.config.Writes()
	if len(writers) == 0 {
		return nil, errors.DatabaseConfigNotFound
	}

	return NewDocker(r.config, writers[0].Database, writers[0].Username, writers[0].Password), nil
}

func (r *Mysql) Gorm() (*gorm.DB, driver.GormQuery, error) {
	if r.db != nil {
		return r.db, NewQuery(), nil
	}

	db, err := NewGorm(r.config, r.log).Build()
	if err != nil {
		return nil, nil, err
	}

	r.db = db

	return db, NewQuery(), nil
}

func (r *Mysql) Grammar() contractsschema.Grammar {
	return NewGrammar(r.config.Writes()[0].Database, r.config.Writes()[0].Prefix)
}

func (r *Mysql) Processor() contractsschema.Processor {
	return NewProcessor()
}

func (r *Mysql) versionAndName() (string, string) {
	version := str.Of(r.version())
	if version.Contains("MariaDB") {
		return version.Between("5.5.5-", "-MariaDB").String(), "MariaDB"
	}
	return version.String(), Name
}

func (r *Mysql) version() string {
	instance, _, err := r.Gorm()
	if err != nil {
		return ""
	}

	var version struct {
		Value string
	}
	if err := instance.Raw("SELECT VERSION() AS value;").Scan(&version).Error; err != nil {
		return fmt.Sprintf("UNKNOWN: %s", err)
	}

	return version.Value
}
