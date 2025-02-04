package mysql

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cast"

	contractsschema "github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/database/schema"
)

var _ contractsschema.Grammar = &Grammar{}

type Grammar struct {
	attributeCommands []string
	database          string
	modifiers         []func(contractsschema.Blueprint, contractsschema.ColumnDefinition) string
	prefix            string
	serials           []string
	wrap              *Wrap
}

func NewGrammar(database, prefix string) *Grammar {
	grammar := &Grammar{
		attributeCommands: []string{schema.CommandComment},
		database:          database,
		prefix:            prefix,
		serials:           []string{"bigInteger", "integer", "mediumInteger", "smallInteger", "tinyInteger"},
		wrap:              NewWrap(prefix),
	}
	grammar.modifiers = []func(contractsschema.Blueprint, contractsschema.ColumnDefinition) string{
		// The sort should not be changed, it effects the SQL output
		grammar.ModifyUnsigned,
		grammar.ModifyNullable,
		grammar.ModifyDefault,
		grammar.ModifyOnUpdate,
		grammar.ModifyIncrement,
		grammar.ModifyComment,
	}

	return grammar
}

func (r *Grammar) CompileAdd(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return fmt.Sprintf("alter table %s add %s", r.wrap.Table(blueprint.GetTableName()), r.getColumn(blueprint, command.Column))
}

func (r *Grammar) CompileChange(blueprint contractsschema.Blueprint, command *contractsschema.Command) []string {
	return []string{
		fmt.Sprintf("alter table %s modify %s", r.wrap.Table(blueprint.GetTableName()), r.getColumn(blueprint, command.Column)),
	}
}

func (r *Grammar) CompileColumns(_, table string) (string, error) {
	table = r.prefix + table

	return fmt.Sprintf(
		"select column_name as `name`, data_type as `type_name`, column_type as `type`, "+
			"collation_name as `collation`, is_nullable as `nullable`, "+
			"column_default as `default`, column_comment as `comment`, "+
			"generation_expression as `expression`, extra as `extra` "+
			"from information_schema.columns where table_schema = %s and table_name = %s "+
			"order by ordinal_position asc", r.wrap.Quote(r.database), r.wrap.Quote(table)), nil
}

func (r *Grammar) CompileComment(_ contractsschema.Blueprint, _ *contractsschema.Command) string {
	return ""
}

func (r *Grammar) CompileCreate(blueprint contractsschema.Blueprint) string {
	columns := r.getColumns(blueprint)
	primaryCommand := getCommandByName(blueprint.GetCommands(), "primary")
	if primaryCommand != nil {
		var algorithm string
		if primaryCommand.Algorithm != "" {
			algorithm = "using " + primaryCommand.Algorithm
		}
		columns = append(columns, fmt.Sprintf("primary key %s(%s)", algorithm, r.wrap.Columnize(primaryCommand.Columns)))

		primaryCommand.ShouldBeSkipped = true
	}

	return fmt.Sprintf("create table %s (%s)", r.wrap.Table(blueprint.GetTableName()), strings.Join(columns, ", "))
}

func (r *Grammar) CompileDefault(_ contractsschema.Blueprint, _ *contractsschema.Command) string {
	return ""
}

func (r *Grammar) CompileDisableForeignKeyConstraints() string {
	return "SET FOREIGN_KEY_CHECKS=0;"
}

func (r *Grammar) CompileDrop(blueprint contractsschema.Blueprint) string {
	return fmt.Sprintf("drop table %s", r.wrap.Table(blueprint.GetTableName()))
}

func (r *Grammar) CompileDropAllDomains(_ []string) string {
	return ""
}

func (r *Grammar) CompileDropAllTables(_ string, tables []contractsschema.Table) []string {
	var dropTables []string
	for _, table := range tables {
		dropTables = append(dropTables, table.Name)
	}

	return []string{
		r.CompileDisableForeignKeyConstraints(),
		fmt.Sprintf("drop table %s", r.wrap.Columnize(dropTables)),
		r.CompileEnableForeignKeyConstraints(),
	}
}

func (r *Grammar) CompileDropAllTypes(_ string, _ []contractsschema.Type) []string {
	return nil
}

func (r *Grammar) CompileDropAllViews(_ string, views []contractsschema.View) []string {
	var dropViews []string
	for _, view := range views {
		dropViews = append(dropViews, view.Name)
	}

	return []string{
		fmt.Sprintf("drop view %s", r.wrap.Columnize(dropViews)),
	}
}

func (r *Grammar) CompileDropColumn(blueprint contractsschema.Blueprint, command *contractsschema.Command) []string {
	columns := r.wrap.PrefixArray("drop", r.wrap.Columns(command.Columns))

	return []string{
		fmt.Sprintf("alter table %s %s", r.wrap.Table(blueprint.GetTableName()), strings.Join(columns, ", ")),
	}
}

func (r *Grammar) CompileDropForeign(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return fmt.Sprintf("alter table %s drop foreign key %s", r.wrap.Table(blueprint.GetTableName()), r.wrap.Column(command.Index))
}

func (r *Grammar) CompileDropFullText(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return r.CompileDropIndex(blueprint, command)
}

func (r *Grammar) CompileDropIfExists(blueprint contractsschema.Blueprint) string {
	return fmt.Sprintf("drop table if exists %s", r.wrap.Table(blueprint.GetTableName()))
}

func (r *Grammar) CompileDropIndex(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return fmt.Sprintf("alter table %s drop index %s", r.wrap.Table(blueprint.GetTableName()), r.wrap.Column(command.Index))
}

func (r *Grammar) CompileDropPrimary(blueprint contractsschema.Blueprint, _ *contractsschema.Command) string {
	return fmt.Sprintf("alter table %s drop primary key", r.wrap.Table(blueprint.GetTableName()))
}

func (r *Grammar) CompileDropUnique(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return r.CompileDropIndex(blueprint, command)
}

func (r *Grammar) CompileEnableForeignKeyConstraints() string {
	return "SET FOREIGN_KEY_CHECKS=1;"
}

func (r *Grammar) CompileForeign(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	sql := fmt.Sprintf("alter table %s add constraint %s foreign key (%s) references %s (%s)",
		r.wrap.Table(blueprint.GetTableName()),
		r.wrap.Column(command.Index),
		r.wrap.Columnize(command.Columns),
		r.wrap.Table(command.On),
		r.wrap.Columnize(command.References))
	if command.OnDelete != "" {
		sql += " on delete " + command.OnDelete
	}
	if command.OnUpdate != "" {
		sql += " on update " + command.OnUpdate
	}

	return sql
}

func (r *Grammar) CompileForeignKeys(_, table string) string {
	return fmt.Sprintf(
		`SELECT 
			kc.constraint_name AS name, 
			GROUP_CONCAT(kc.column_name ORDER BY kc.ordinal_position) AS columns, 
			kc.referenced_table_schema AS foreign_schema, 
			kc.referenced_table_name AS foreign_table, 
			GROUP_CONCAT(kc.referenced_column_name ORDER BY kc.ordinal_position) AS foreign_columns, 
			rc.update_rule AS on_update, 
			rc.delete_rule AS on_delete 
		FROM information_schema.key_column_usage kc 
		JOIN information_schema.referential_constraints rc 
			ON kc.constraint_schema = rc.constraint_schema 
			AND kc.constraint_name = rc.constraint_name 
		WHERE kc.table_schema = %s 
			AND kc.table_name = %s 
			AND kc.referenced_table_name IS NOT NULL 
		GROUP BY 
			kc.constraint_name, 
			kc.referenced_table_schema, 
			kc.referenced_table_name, 
			rc.update_rule, 
			rc.delete_rule`,
		r.wrap.Quote(r.database),
		r.wrap.Quote(table),
	)
}

func (r *Grammar) CompileFullText(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return r.compileKey(blueprint, command, "fulltext")
}

func (r *Grammar) CompileIndex(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	var algorithm string
	if command.Algorithm != "" {
		algorithm = " using " + command.Algorithm
	}

	return fmt.Sprintf("alter table %s add %s %s%s(%s)",
		r.wrap.Table(blueprint.GetTableName()),
		"index",
		r.wrap.Column(command.Index),
		algorithm,
		r.wrap.Columnize(command.Columns),
	)
}

func (r *Grammar) CompileIndexes(_, table string) (string, error) {
	table = r.prefix + table

	return fmt.Sprintf(
		"select index_name as `name`, group_concat(column_name order by seq_in_index) as `columns`, "+
			"index_type as `type`, not non_unique as `unique` "+
			"from information_schema.statistics where table_schema = %s and table_name = %s "+
			"group by index_name, index_type, non_unique",
		r.wrap.Quote(r.database),
		r.wrap.Quote(table),
	), nil
}

func (r *Grammar) CompilePrimary(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	var algorithm string
	if command.Algorithm != "" {
		algorithm = "using " + command.Algorithm
	}

	return fmt.Sprintf("alter table %s add primary key %s(%s)", r.wrap.Table(blueprint.GetTableName()), algorithm, r.wrap.Columnize(command.Columns))
}

func (r *Grammar) CompileRename(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return fmt.Sprintf("rename table %s to %s", r.wrap.Table(blueprint.GetTableName()), r.wrap.Table(command.To))
}

func (r *Grammar) CompileRenameIndex(_ contractsschema.Schema, blueprint contractsschema.Blueprint, command *contractsschema.Command) []string {
	return []string{
		fmt.Sprintf("alter table %s rename index %s to %s", r.wrap.Table(blueprint.GetTableName()), r.wrap.Column(command.From), r.wrap.Column(command.To)),
	}
}

func (r *Grammar) CompileTables(database string) string {
	return fmt.Sprintf("select table_name as `name`, (data_length + index_length) as `size`, "+
		"table_comment as `comment`, engine as `engine`, table_collation as `collation` "+
		"from information_schema.tables where table_schema = %s and table_type in ('BASE TABLE', 'SYSTEM VERSIONED') "+
		"order by table_name", r.wrap.Quote(database))
}

func (r *Grammar) CompileTypes() string {
	return ""
}

func (r *Grammar) CompileUnique(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return r.compileKey(blueprint, command, "unique")
}

func (r *Grammar) CompileViews(database string) string {
	return fmt.Sprintf("select table_name as `name`, view_definition as `definition` "+
		"from information_schema.views where table_schema = %s "+
		"order by table_name", r.wrap.Quote(database))
}

func (r *Grammar) GetAttributeCommands() []string {
	return r.attributeCommands
}

func (r *Grammar) ModifyComment(_ contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	if comment := column.GetComment(); comment != "" {
		// Escape special characters to prevent SQL injection
		comment = strings.ReplaceAll(comment, "'", "''")
		comment = strings.ReplaceAll(comment, "\\", "\\\\")

		return fmt.Sprintf(" comment '%s'", comment)
	}

	return ""
}

func (r *Grammar) ModifyDefault(_ contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	if column.GetDefault() != nil {
		return fmt.Sprintf(" default %s", schema.ColumnDefaultValue(column.GetDefault()))
	}

	return ""
}

func (r *Grammar) ModifyNullable(_ contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	if column.GetNullable() {
		return " null"
	} else {
		return " not null"
	}
}

func (r *Grammar) ModifyIncrement(blueprint contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	if slices.Contains(r.serials, column.GetType()) && column.GetAutoIncrement() {
		if blueprint.HasCommand("primary") {
			return "auto_increment"
		}
		return " auto_increment primary key"
	}

	return ""
}

func (r *Grammar) ModifyOnUpdate(_ contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	onUpdate := column.GetOnUpdate()
	if onUpdate != nil {
		switch value := onUpdate.(type) {
		case schema.Expression:
			return " on update " + string(value)
		case string:
			if onUpdate.(string) != "" {
				return " on update " + value
			}
		}
	}

	return ""
}

func (r *Grammar) ModifyUnsigned(_ contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	if column.GetUnsigned() {
		return " unsigned"
	}

	return ""
}

func (r *Grammar) TypeBigInteger(_ contractsschema.ColumnDefinition) string {
	return "bigint"
}

func (r *Grammar) TypeBoolean(_ contractsschema.ColumnDefinition) string {
	return "tinyint(1)"
}

func (r *Grammar) TypeChar(column contractsschema.ColumnDefinition) string {
	return fmt.Sprintf("char(%d)", column.GetLength())
}

func (r *Grammar) TypeDate(_ contractsschema.ColumnDefinition) string {
	return "date"
}

func (r *Grammar) TypeDateTime(column contractsschema.ColumnDefinition) string {
	current := "CURRENT_TIMESTAMP"
	precision := column.GetPrecision()
	if precision > 0 {
		current = fmt.Sprintf("CURRENT_TIMESTAMP(%d)", precision)
	}
	if column.GetUseCurrent() {
		column.Default(schema.Expression(current))
	}
	if column.GetUseCurrentOnUpdate() {
		column.OnUpdate(schema.Expression(current))
	}

	if precision > 0 {
		return fmt.Sprintf("datetime(%d)", precision)
	} else {
		return "datetime"
	}
}

func (r *Grammar) TypeDateTimeTz(column contractsschema.ColumnDefinition) string {
	return r.TypeDateTime(column)
}

func (r *Grammar) TypeDecimal(column contractsschema.ColumnDefinition) string {
	return fmt.Sprintf("decimal(%d, %d)", column.GetTotal(), column.GetPlaces())
}

func (r *Grammar) TypeDouble(_ contractsschema.ColumnDefinition) string {
	return "double"
}

func (r *Grammar) TypeEnum(column contractsschema.ColumnDefinition) string {
	return fmt.Sprintf(`enum(%s)`, strings.Join(r.wrap.Quotes(cast.ToStringSlice(column.GetAllowed())), ", "))
}

func (r *Grammar) TypeFloat(column contractsschema.ColumnDefinition) string {
	precision := column.GetPrecision()
	if precision > 0 {
		return fmt.Sprintf("float(%d)", precision)
	}

	return "float"
}

func (r *Grammar) TypeInteger(_ contractsschema.ColumnDefinition) string {
	return "int"
}

func (r *Grammar) TypeJson(_ contractsschema.ColumnDefinition) string {
	return "json"
}

func (r *Grammar) TypeJsonb(_ contractsschema.ColumnDefinition) string {
	return "json"
}

func (r *Grammar) TypeLongText(_ contractsschema.ColumnDefinition) string {
	return "longtext"
}

func (r *Grammar) TypeMediumInteger(_ contractsschema.ColumnDefinition) string {
	return "mediumint"
}

func (r *Grammar) TypeMediumText(_ contractsschema.ColumnDefinition) string {
	return "mediumtext"
}

func (r *Grammar) TypeSmallInteger(_ contractsschema.ColumnDefinition) string {
	return "smallint"
}

func (r *Grammar) TypeString(column contractsschema.ColumnDefinition) string {
	length := column.GetLength()
	if length > 0 {
		return fmt.Sprintf("varchar(%d)", length)
	}

	return "varchar(255)"
}

func (r *Grammar) TypeText(_ contractsschema.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeTime(column contractsschema.ColumnDefinition) string {
	if column.GetPrecision() > 0 {
		return fmt.Sprintf("time(%d)", column.GetPrecision())
	} else {
		return "time"
	}
}

func (r *Grammar) TypeTimeTz(column contractsschema.ColumnDefinition) string {
	return r.TypeTime(column)
}

func (r *Grammar) TypeTimestamp(column contractsschema.ColumnDefinition) string {
	current := "CURRENT_TIMESTAMP"
	precision := column.GetPrecision()
	if precision > 0 {
		current = fmt.Sprintf("CURRENT_TIMESTAMP(%d)", precision)
	}
	if column.GetUseCurrent() {
		column.Default(schema.Expression(current))
	}
	if column.GetUseCurrentOnUpdate() {
		column.OnUpdate(schema.Expression(current))
	}

	if precision > 0 {
		return fmt.Sprintf("timestamp(%d)", precision)
	} else {
		return "timestamp"
	}
}

func (r *Grammar) TypeTimestampTz(column contractsschema.ColumnDefinition) string {
	return r.TypeTimestamp(column)
}

func (r *Grammar) TypeTinyInteger(_ contractsschema.ColumnDefinition) string {
	return "tinyint"
}

func (r *Grammar) TypeTinyText(_ contractsschema.ColumnDefinition) string {
	return "tinytext"
}

func (r *Grammar) compileKey(blueprint contractsschema.Blueprint, command *contractsschema.Command, ttype string) string {
	var algorithm string
	if command.Algorithm != "" {
		algorithm = " using " + command.Algorithm
	}

	return fmt.Sprintf("alter table %s add %s %s%s(%s)",
		r.wrap.Table(blueprint.GetTableName()),
		ttype,
		r.wrap.Column(command.Index),
		algorithm,
		r.wrap.Columnize(command.Columns))
}

func (r *Grammar) getColumns(blueprint contractsschema.Blueprint) []string {
	var columns []string
	for _, column := range blueprint.GetAddedColumns() {
		columns = append(columns, r.getColumn(blueprint, column))
	}

	return columns
}

func (r *Grammar) getColumn(blueprint contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	sql := fmt.Sprintf("%s %s", r.wrap.Column(column.GetName()), schema.ColumnType(r, column))

	for _, modifier := range r.modifiers {
		sql += modifier(blueprint, column)
	}

	return sql
}

func getCommandByName(commands []*contractsschema.Command, name string) *contractsschema.Command {
	commands = getCommandsByName(commands, name)
	if len(commands) == 0 {
		return nil
	}

	return commands[0]
}

func getCommandsByName(commands []*contractsschema.Command, name string) []*contractsschema.Command {
	var filteredCommands []*contractsschema.Command
	for _, command := range commands {
		if command.Name == name {
			filteredCommands = append(filteredCommands, command)
		}
	}

	return filteredCommands
}
