package mysql

import (
	"fmt"

	"github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/contracts/database"
	"github.com/goravel/framework/contracts/database/driver"
	contractsschema "github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/contracts/log"
	"github.com/goravel/framework/contracts/testing"
	"github.com/goravel/framework/errors"
	"github.com/goravel/mysql/contracts"
	"gorm.io/gorm"
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

	return database.Config{
		Connection: writers[0].Connection,
		Database:   writers[0].Database,
		Driver:     Name,
		Host:       writers[0].Host,
		Password:   writers[0].Password,
		Port:       writers[0].Port,
		Prefix:     writers[0].Prefix,
		Username:   writers[0].Username,
		Version:    r.version(),
	}
}

func (r *Mysql) Docker() (testing.DatabaseDriver, error) {
	writers := r.config.Writes()
	if len(writers) == 0 {
		return nil, errors.OrmDatabaseConfigNotFound
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
	return NewGrammar(r.config.Writes()[0].Prefix)
}

func (r *Mysql) Processor() contractsschema.Processor {
	return NewProcessor()
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
