package mysql

import (
	"fmt"
	"strconv"
	"time"

	contractsprocess "github.com/goravel/framework/contracts/process"
	contractsdocker "github.com/goravel/framework/contracts/testing/docker"
	"github.com/goravel/framework/support/color"
	supportdocker "github.com/goravel/framework/support/docker"
	testingdocker "github.com/goravel/framework/testing/docker"
	"github.com/spf13/cast"
	"gorm.io/driver/mysql"
	gormio "gorm.io/gorm"

	"github.com/goravel/mysql/contracts"
)

type Docker struct {
	config         contracts.ConfigBuilder
	databaseConfig contractsdocker.DatabaseConfig
	imageDriver    contractsdocker.ImageDriver
	process        contractsprocess.Process
}

func NewDocker(config contracts.ConfigBuilder, process contractsprocess.Process, database, username, password string) *Docker {
	env := []string{
		"MYSQL_ROOT_PASSWORD=" + password,
		"MYSQL_DATABASE=" + database,
	}
	if username != "root" {
		env = append(env, "MYSQL_USER="+username)
		env = append(env, "MYSQL_PASSWORD="+password)
	}

	return &Docker{
		config: config,
		databaseConfig: contractsdocker.DatabaseConfig{
			Driver:   Name,
			Host:     "127.0.0.1",
			Port:     3306,
			Database: database,
			Username: username,
			Password: password,
		},
		imageDriver: testingdocker.NewImageDriver(contractsdocker.Image{
			Repository:   "mysql",
			Tag:          "latest",
			Env:          env,
			ExposedPorts: []string{"3306"},
		}, process),
		process: process,
	}
}

func (r *Docker) Build() error {
	if err := r.imageDriver.Build(); err != nil {
		return err
	}

	config := r.imageDriver.Config()
	r.databaseConfig.ContainerID = config.ContainerID
	r.databaseConfig.Port = cast.ToInt(supportdocker.ExposedPort(config.ExposedPorts, strconv.Itoa(r.databaseConfig.Port)))

	return nil
}

func (r *Docker) Config() contractsdocker.DatabaseConfig {
	return r.databaseConfig
}

func (r *Docker) Database(name string) (contractsdocker.DatabaseDriver, error) {
	go func() {
		instance, err := r.connect("root")
		if err != nil {
			color.Errorf("connect Mysql error: %v", err)
			return
		}

		res := instance.Exec(fmt.Sprintf(`CREATE DATABASE %s;`, name))
		if res.Error != nil {
			color.Errorf("create Mysql database error: %v", res.Error)
			return
		}

		res = instance.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO `%s`@`%%`;", name, r.databaseConfig.Username))
		if res.Error != nil {
			color.Errorf("grant privileges in Mysql database error: %v", res.Error)
		}

		if err := r.close(instance); err != nil {
			color.Errorf("close Mysql connection error: %v", err)
		}
	}()

	docker := NewDocker(r.config, r.process, name, r.databaseConfig.Username, r.databaseConfig.Password)
	docker.databaseConfig.ContainerID = r.databaseConfig.ContainerID
	docker.databaseConfig.Port = r.databaseConfig.Port

	return docker, nil
}

func (r *Docker) Driver() string {
	return Name
}

func (r *Docker) Fresh() error {
	instance, err := r.connect()
	if err != nil {
		return fmt.Errorf("connect Mysql error when clearing: %v", err)
	}

	res := instance.Raw("select concat('drop table ',table_name,';') from information_schema.TABLES where table_schema=?;", r.databaseConfig.Database)
	if res.Error != nil {
		return fmt.Errorf("get tables of Mysql error: %v", res.Error)
	}

	var tables []string
	res = res.Scan(&tables)
	if res.Error != nil {
		return fmt.Errorf("get tables of Mysql error: %v", res.Error)
	}

	if res := instance.Exec("SET FOREIGN_KEY_CHECKS=0;"); res.Error != nil {
		return fmt.Errorf("disable foreign key check of Mysql error: %v", res.Error)
	}

	for _, table := range tables {
		res = instance.Exec(table)
		if res.Error != nil {
			return fmt.Errorf("drop table %s of Mysql error: %v", table, res.Error)
		}
	}

	if res := instance.Exec("SET FOREIGN_KEY_CHECKS=1;"); res.Error != nil {
		return fmt.Errorf("enable foreign key check of Mysql error: %v", res.Error)
	}

	return r.close(instance)
}

func (r *Docker) Image(image contractsdocker.Image) {
	r.imageDriver = testingdocker.NewImageDriver(image, r.process)
}

func (r *Docker) Ready() error {
	gormDB, err := r.connect()
	if err != nil {
		return err
	}

	r.resetConfigPort()

	return r.close(gormDB)
}

func (r *Docker) Reuse(containerID string, port int) error {
	r.databaseConfig.ContainerID = containerID
	r.databaseConfig.Port = port

	return nil
}

func (r *Docker) Shutdown() error {
	return r.imageDriver.Shutdown()
}

func (r *Docker) connect(username ...string) (*gormio.DB, error) {
	var (
		instance *gormio.DB
		err      error
	)

	useUsername := r.databaseConfig.Username
	if len(username) > 0 {
		useUsername = username[0]
	}

	// docker compose need time to start
	for i := 0; i < 60; i++ {
		instance, err = gormio.Open(mysql.New(mysql.Config{
			DSN: fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", useUsername, r.databaseConfig.Password, r.databaseConfig.Host, r.databaseConfig.Port, r.databaseConfig.Database),
		}))

		if err == nil {
			break
		}

		time.Sleep(1 * time.Second)
	}

	return instance, err
}

func (r *Docker) close(gormDB *gormio.DB) error {
	db, err := gormDB.DB()
	if err != nil {
		return err
	}

	return db.Close()
}

func (r *Docker) resetConfigPort() {
	writers := r.config.Config().Get(fmt.Sprintf("database.connections.%s.write", r.config.Connection()))
	if writeConfigs, ok := writers.([]contracts.Config); ok {
		writeConfigs[0].Port = r.databaseConfig.Port
		r.config.Config().Add(fmt.Sprintf("database.connections.%s.write", r.config.Connection()), writeConfigs)

		return
	}

	r.config.Config().Add(fmt.Sprintf("database.connections.%s.port", r.config.Connection()), r.databaseConfig.Port)
}
