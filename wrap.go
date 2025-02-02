package mysql

import (
	"strings"

	"github.com/goravel/framework/database/schema"
)

type Wrap struct {
	*schema.Wrap
	prefix string
}

func NewWrap(prefix string) *Wrap {
	return &Wrap{
		Wrap:   schema.NewWrap(prefix),
		prefix: prefix,
	}
}

func (r *Wrap) Value(value string) string {
	if value != "*" {
		return "`" + strings.ReplaceAll(value, "`", "``") + "`"
	}

	return value
}
