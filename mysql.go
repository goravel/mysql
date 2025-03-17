package mysql

import (
	"fmt"

	"github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/contracts/database"
	contractsdriver "github.com/goravel/framework/contracts/database/driver"
	"github.com/goravel/framework/contracts/log"
	"github.com/goravel/framework/contracts/testing/docker"
	"github.com/goravel/framework/database/driver"
	"github.com/goravel/framework/errors"
	"github.com/goravel/framework/support/str"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"

	"github.com/goravel/mysql/contracts"
)

var _ contractsdriver.Driver = &Mysql{}

type Mysql struct {
	config  contracts.ConfigBuilder
	log     log.Log
	version string
}

func NewMysql(config config.Config, log log.Log, connection string) *Mysql {
	return &Mysql{
		config: NewConfig(config, connection),
		log:    log,
	}
}

func (r *Mysql) Docker() (docker.DatabaseDriver, error) {
	writers := r.config.Writers()
	if len(writers) == 0 {
		return nil, errors.DatabaseConfigNotFound
	}

	return NewDocker(r.config, writers[0].Database, writers[0].Username, writers[0].Password), nil
}

func (r *Mysql) Grammar() contractsdriver.Grammar {
	version, name := r.versionAndName()

	return NewGrammar(r.config.Writers()[0].Database, r.config.Writers()[0].Prefix, version, name)
}

func (r *Mysql) Pool() database.Pool {
	return database.Pool{
		Readers: r.fullConfigsToConfigs(r.config.Readers()),
		Writers: r.fullConfigsToConfigs(r.config.Writers()),
	}
}

func (r *Mysql) Processor() contractsdriver.Processor {
	return NewProcessor()
}

func (r *Mysql) fullConfigsToConfigs(fullConfigs []contracts.FullConfig) []database.Config {
	configs := make([]database.Config, len(fullConfigs))
	for i, fullConfig := range fullConfigs {
		configs[i] = database.Config{
			Connection:   fullConfig.Connection,
			Dsn:          fullConfig.Dsn,
			Database:     fullConfig.Database,
			Dialector:    fullConfigToDialector(fullConfig),
			Driver:       Name,
			Host:         fullConfig.Host,
			NameReplacer: fullConfig.NameReplacer,
			NoLowerCase:  fullConfig.NoLowerCase,
			Password:     fullConfig.Password,
			Port:         fullConfig.Port,
			Prefix:       fullConfig.Prefix,
			Singular:     fullConfig.Singular,
			Username:     fullConfig.Username,
		}
	}

	return configs
}

func (r *Mysql) versionAndName() (string, string) {
	version := str.Of(r.getVersion())
	if version.Contains("MariaDB") {
		return version.Between("5.5.5-", "-MariaDB").String(), "MariaDB"
	}
	return version.String(), Name
}

func (r *Mysql) getVersion() string {
	if r.version != "" {
		return r.version
	}

	instance, err := driver.BuildGorm(r.config.Config(), r.log, r.Pool())
	if err != nil {
		return ""
	}

	var version struct {
		Value string
	}
	if err := instance.Clauses(dbresolver.Write).Raw(r.Grammar().CompileVersion()).Scan(&version).Error; err != nil {
		r.version = fmt.Sprintf("UNKNOWN: %s", err)
	} else {
		r.version = version.Value
	}

	return r.version
}

func dsn(fullConfig contracts.FullConfig) string {
	if fullConfig.Dsn != "" {
		return fullConfig.Dsn
	}
	if fullConfig.Host == "" {
		return ""
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s&multiStatements=true",
		fullConfig.Username, fullConfig.Password, fullConfig.Host, fullConfig.Port, fullConfig.Database, fullConfig.Charset, true, fullConfig.Loc)
}

func fullConfigToDialector(fullConfig contracts.FullConfig) gorm.Dialector {
	dsn := dsn(fullConfig)
	if dsn == "" {
		return nil
	}

	return mysql.New(mysql.Config{
		DSN: dsn,
	})
}
