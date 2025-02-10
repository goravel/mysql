package mysql

import (
	"strings"

	"github.com/goravel/framework/contracts/database/schema"
)

var _ schema.Processor = &Processor{}

type Processor struct {
}

func NewProcessor() *Processor {
	return &Processor{}
}

func (r Processor) ProcessColumns(dbColumns []schema.DBColumn) []schema.Column {
	var columns []schema.Column
	for _, dbColumn := range dbColumns {
		var nullable bool
		if dbColumn.Nullable == "YES" {
			nullable = true
		}
		var autoIncrement bool
		if dbColumn.Extra == "auto_increment" {
			autoIncrement = true
		}

		columns = append(columns, schema.Column{
			Autoincrement: autoIncrement,
			Collation:     dbColumn.Collation,
			Comment:       dbColumn.Comment,
			Default:       dbColumn.Default,
			Extra:         dbColumn.Extra,
			Name:          dbColumn.Name,
			Nullable:      nullable,
			Type:          dbColumn.Type,
			TypeName:      dbColumn.TypeName,
		})
	}

	return columns
}

func (r Processor) ProcessForeignKeys(dbForeignKeys []schema.DBForeignKey) []schema.ForeignKey {
	var foreignKeys []schema.ForeignKey
	for _, dbForeignKey := range dbForeignKeys {
		foreignKeys = append(foreignKeys, schema.ForeignKey{
			Name:           dbForeignKey.Name,
			Columns:        strings.Split(dbForeignKey.Columns, ","),
			ForeignSchema:  dbForeignKey.ForeignSchema,
			ForeignTable:   dbForeignKey.ForeignTable,
			ForeignColumns: strings.Split(dbForeignKey.ForeignColumns, ","),
			OnUpdate:       strings.ToLower(dbForeignKey.OnUpdate),
			OnDelete:       strings.ToLower(dbForeignKey.OnDelete),
		})
	}

	return foreignKeys
}

func (r Processor) ProcessIndexes(dbIndexes []schema.DBIndex) []schema.Index {
	var indexes []schema.Index
	for _, dbIndex := range dbIndexes {
		name := strings.ToLower(dbIndex.Name)
		indexes = append(indexes, schema.Index{
			Columns: strings.Split(dbIndex.Columns, ","),
			Name:    name,
			Type:    strings.ToLower(dbIndex.Type),
			Primary: name == "primary",
			Unique:  dbIndex.Unique,
		})
	}

	return indexes
}

func (r Processor) ProcessTypes(types []schema.Type) []schema.Type {
	return types
}
