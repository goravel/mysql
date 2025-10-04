package main

import (
	"os"

	"github.com/goravel/framework/packages"
	"github.com/goravel/framework/packages/match"
	"github.com/goravel/framework/packages/modify"
	"github.com/goravel/framework/support/path"
)

var config = `map[string]any{
        "host":     config.Env("DB_HOST", "127.0.0.1"),
        "port":     config.Env("DB_PORT", 3306),
        "database": config.Env("DB_DATABASE", "forge"),
        "username": config.Env("DB_USERNAME", ""),
        "password": config.Env("DB_PASSWORD", ""),
        "charset":  "utf8mb4",
        "prefix":   "",
        "singular": false,
        "via": func() (driver.Driver, error) {
            return mysqlfacades.Mysql("mysql")
        },
    }`

func main() {
	appConfigPath := path.Config("app.go")
	databaseConfigPath := path.Config("database.go")
	modulePath := packages.GetModulePath()
	mysqlServiceProvider := "&mysql.ServiceProvider{}"
	driverContract := "github.com/goravel/framework/contracts/database/driver"
	mysqlFacades := "github.com/goravel/mysql/facades"

	packages.Setup(os.Args).
		Install(
			// Add mysql service provider to app.go
			modify.GoFile(appConfigPath).
				Find(match.Imports()).Modify(modify.AddImport(modulePath)).
				Find(match.Providers()).Modify(modify.Register(mysqlServiceProvider)),

			// Add mysql connection config to database.go
			modify.GoFile(databaseConfigPath).
				Find(match.Imports()).Modify(
				modify.AddImport(driverContract),
				modify.AddImport(mysqlFacades, "mysqlfacades"),
			).
				Find(match.Config("database.connections")).Modify(modify.AddConfig("mysql", config)).
				Find(match.Config("database")).Modify(modify.AddConfig("default", `"mysql"`)),
		).
		Uninstall(
			// Remove mysql connection from database.go
			modify.GoFile(databaseConfigPath).
				Find(match.Config("database")).Modify(modify.AddConfig("default", `""`)).
				Find(match.Config("database.connections")).Modify(modify.RemoveConfig("mysql")).
				Find(match.Imports()).Modify(modify.RemoveImport(driverContract), modify.RemoveImport(mysqlFacades, "mysqlfacades")),

			// Remove mysql service provider from app.go
			modify.GoFile(appConfigPath).
				Find(match.Providers()).Modify(modify.Unregister(mysqlServiceProvider)).
				Find(match.Imports()).Modify(modify.RemoveImport(modulePath)),
		).
		Execute()
}
