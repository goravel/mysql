package mysql

import (
	"fmt"
	"time"

	contractsdocker "github.com/goravel/framework/contracts/testing/docker"
	"github.com/goravel/framework/support/color"
	"github.com/goravel/framework/support/docker"
	"github.com/goravel/framework/support/process"
	"github.com/goravel/mysql/contracts"
	"gorm.io/driver/mysql"
	gormio "gorm.io/gorm"
)

type Docker struct {
	config      contracts.ConfigBuilder
	containerID string
	database    string
	host        string
	image       *contractsdocker.Image
	password    string
	username    string
	port        int
}

func NewDocker(config contracts.ConfigBuilder, database, username, password string) *Docker {
	env := []string{
		"MYSQL_ROOT_PASSWORD=" + password,
		"MYSQL_DATABASE=" + database,
	}
	if username != "root" {
		env = append(env, "MYSQL_USER="+username)
		env = append(env, "MYSQL_PASSWORD="+password)
	}

	return &Docker{
		config:   config,
		database: database,
		host:     "127.0.0.1",
		username: username,
		password: password,
		image: &contractsdocker.Image{
			Repository:   "mysql",
			Tag:          "latest",
			Env:          env,
			ExposedPorts: []string{"3306"},
		},
	}
}

func (r *Docker) Build() error {
	command, exposedPorts := docker.ImageToCommand(r.image)
	containerID, err := process.Run(command)
	if err != nil {
		return fmt.Errorf("init MySQL error: %v", err)
	}
	if containerID == "" {
		return fmt.Errorf("no container id return when creating MySQL docker")
	}

	r.containerID = containerID
	r.port = docker.ExposedPort(exposedPorts, 3306)

	return nil
}

func (r *Docker) Config() contractsdocker.DatabaseConfig {
	return contractsdocker.DatabaseConfig{
		ContainerID: r.containerID,
		Driver:      Name,
		Host:        r.host,
		Port:        r.port,
		Database:    r.database,
		Username:    r.username,
		Password:    r.password,
	}
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

		res = instance.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO `%s`@`%%`;", name, r.username))
		if res.Error != nil {
			color.Errorf("grant privileges in Mysql database error: %v", res.Error)
		}

		if err := r.close(instance); err != nil {
			color.Errorf("close Mysql connection error: %v", err)
		}
	}()

	docker := NewDocker(r.config, name, r.username, r.password)
	docker.containerID = r.containerID
	docker.port = r.port

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

	res := instance.Raw("select concat('drop table ',table_name,';') from information_schema.TABLES where table_schema=?;", r.database)
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
	r.image = &image
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
	r.containerID = containerID
	r.port = port

	return nil
}

func (r *Docker) Shutdown() error {
	if _, err := process.Run(fmt.Sprintf("docker stop %s", r.containerID)); err != nil {
		return fmt.Errorf("stop Mysql error: %v", err)
	}

	return nil
}

func (r *Docker) connect(username ...string) (*gormio.DB, error) {
	var (
		instance *gormio.DB
		err      error
	)

	useUsername := r.username
	if len(username) > 0 {
		useUsername = username[0]
	}

	// docker compose need time to start
	for i := 0; i < 60; i++ {
		instance, err = gormio.Open(mysql.New(mysql.Config{
			DSN: fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", useUsername, r.password, r.host, r.port, r.database),
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
		writeConfigs[0].Port = r.port
		r.config.Config().Add(fmt.Sprintf("database.connections.%s.write", r.config.Connection()), writeConfigs)

		return
	}

	r.config.Config().Add(fmt.Sprintf("database.connections.%s.port", r.config.Connection()), r.port)
}
