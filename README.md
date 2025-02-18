# Mysql

The Mysql driver for facades.Orm() of Goravel.

## Version

| goravel/mysql | goravel/framework |
|------------------|-------------------|
| v1.0.*          | v1.16.*           |

## Install

1. Add package

```
go get -u github.com/goravel/mysql
```

2. Register service provider

```
// config/app.go
import "github.com/goravel/mysql"

"providers": []foundation.ServiceProvider{
    ...
    &mysql.ServiceProvider{},
}
```

3. Add Mysql driver to `config/database.go` file

```
// config/database.go
import (
    "github.com/goravel/framework/contracts/database/driver"
    "github.com/goravel/mysql/contracts"
    mysqlfacades "github.com/goravel/mysql/facades"
)

"connections": map[string]any{
    ...
    "mysql": map[string]any{
        "host":     config.Env("DB_HOST", "127.0.0.1"),
        "port":     config.Env("DB_PORT", 3306),
        "database": config.Env("DB_DATABASE", "forge"),
        "username": config.Env("DB_USERNAME", ""),
        "password": config.Env("DB_PASSWORD", ""),
        "charset":  "utf8mb4",
        "loc":      "UTC", // Asia/Shanghai
        "prefix":   "",
        "singular": false,
        "via": func() (driver.Driver, error) {
            return mysqlfacades.Mysql("mysql")
        },
        // Optional
        "read": []contracts.Config{
            {Host: "192.168.1.1", Port: 3306, Database: "forge", Username: "root", Password: "123123"},
        },
        // Optional
        "write": []contracts.Config{
            {Host: "192.168.1.2", Port: 3306, Database: "forge", Username: "root", Password: "123123"},
        },
    },
}
```
