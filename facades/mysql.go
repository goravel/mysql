package facades

import (
	"fmt"

	"github.com/goravel/framework/contracts/database/driver"

	"github.com/goravel/mysql"
)

func Mysql(connection string) (driver.Driver, error) {
	if mysql.App == nil {
		return nil, fmt.Errorf("please register Mysql service provider")
	}

	instance, err := mysql.App.MakeWith(mysql.Binding, map[string]any{
		"connection": connection,
	})
	if err != nil {

		return nil, err
	}

	return instance.(*mysql.Mysql), nil
}
