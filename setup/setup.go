package main

import (
	"os"

	"github.com/goravel/framework/packages"
	"github.com/goravel/framework/packages/match"
	"github.com/goravel/framework/packages/modify"
	"github.com/goravel/framework/support/env"
	"github.com/goravel/framework/support/path"
)

func main() {
	setup := packages.Setup(os.Args)
	config := `map[string]any{
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

	appConfigPath := path.Config("app.go")
	databaseConfigPath := path.Config("database.go")
	moduleImport := setup.Paths().Module().Import()
	mysqlServiceProvider := "&mysql.ServiceProvider{}"
	driverContract := "github.com/goravel/framework/contracts/database/driver"
	mysqlFacades := "github.com/goravel/mysql/facades"
	databaseConnectionsConfig := match.Config("database.connections")
	databaseConfig := match.Config("database")

	setup.Install(
		// Add mysql service provider to app.go if not using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return !env.IsBootstrapSetup()
		}, modify.GoFile(appConfigPath).
			Find(match.Imports()).Modify(modify.AddImport(moduleImport)).
			Find(match.Providers()).Modify(modify.Register(mysqlServiceProvider))),

		// Add mysql service provider to providers.go if using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return env.IsBootstrapSetup()
		}, modify.AddProviderApply(moduleImport, mysqlServiceProvider)),

		// Add mysql connection config to database.go
		modify.GoFile(databaseConfigPath).
			Find(match.Imports()).Modify(
			modify.AddImport(driverContract),
			modify.AddImport(mysqlFacades, "mysqlfacades"),
		).
			Find(databaseConnectionsConfig).Modify(modify.AddConfig("mysql", config)).
			Find(databaseConfig).Modify(modify.AddConfig("default", `"mysql"`)),
	).Uninstall(
		// Remove mysql connection from database.go
		modify.GoFile(databaseConfigPath).
			Find(databaseConfig).Modify(modify.AddConfig("default", `""`)).
			Find(databaseConnectionsConfig).Modify(modify.RemoveConfig("mysql")).
			Find(match.Imports()).Modify(
			modify.RemoveImport(driverContract),
			modify.RemoveImport(mysqlFacades, "mysqlfacades"),
		),

		// Remove mysql service provider from app.go if not using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return !env.IsBootstrapSetup()
		}, modify.GoFile(appConfigPath).
			Find(match.Providers()).Modify(modify.Unregister(mysqlServiceProvider)).
			Find(match.Imports()).Modify(modify.RemoveImport(moduleImport))),

		// Remove mysql service provider from providers.go if using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return env.IsBootstrapSetup()
		}, modify.RemoveProviderApply(moduleImport, mysqlServiceProvider)),
	).Execute()
}
