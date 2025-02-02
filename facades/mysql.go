package facades

import (
	"log"

	"github.com/goravel/framework/contracts/database/driver"

	"github.com/goravel/mysql"
)

func Mysql(connection string) driver.Driver {
	if mysql.App == nil {
		log.Fatalln("please register Mysql service provider")
		return nil
	}

	instance, err := mysql.App.MakeWith(mysql.Binding, map[string]any{
		"connection": connection,
	})
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return instance.(*mysql.Mysql)
}
