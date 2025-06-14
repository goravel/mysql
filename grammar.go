package mysql

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	sq "github.com/Masterminds/squirrel"
	"github.com/goravel/framework/contracts/database/driver"
	"github.com/goravel/framework/database/schema"
	"github.com/goravel/framework/errors"
	"github.com/goravel/framework/support/collect"
	"github.com/spf13/cast"
	"gorm.io/gorm/clause"
)

var _ driver.Grammar = &Grammar{}

type Grammar struct {
	attributeCommands []string
	database          string
	modifiers         []func(driver.Blueprint, driver.ColumnDefinition) string
	name              string
	prefix            string
	serials           []string
	version           string
	wrap              *schema.Wrap
}

func NewGrammar(database, prefix, version, name string) *Grammar {
	grammar := &Grammar{
		attributeCommands: []string{schema.CommandComment},
		database:          database,
		name:              name,
		prefix:            prefix,
		serials:           []string{"bigInteger", "integer", "mediumInteger", "smallInteger", "tinyInteger"},
		version:           version,
		wrap:              schema.NewWrap(prefix),
	}
	grammar.wrap.SetValueWrapper(func(s string) string {
		return "`" + strings.ReplaceAll(s, "`", "``") + "`"
	})
	grammar.modifiers = []func(driver.Blueprint, driver.ColumnDefinition) string{
		// The sort should not be changed, it effects the SQL output
		grammar.ModifyUnsigned,
		grammar.ModifyNullable,
		grammar.ModifyDefault,
		grammar.ModifyOnUpdate,
		grammar.ModifyIncrement,
		grammar.ModifyComment,
		grammar.ModifyAfter,
		grammar.ModifyFirst,
	}

	return grammar
}

func (r *Grammar) CompileAdd(blueprint driver.Blueprint, command *driver.Command) string {
	return fmt.Sprintf("alter table %s add %s", r.wrap.Table(blueprint.GetTableName()), r.getColumn(blueprint, command.Column))
}

func (r *Grammar) CompileChange(blueprint driver.Blueprint, command *driver.Command) []string {
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

func (r *Grammar) CompileComment(_ driver.Blueprint, _ *driver.Command) string {
	return ""
}

func (r *Grammar) CompileCreate(blueprint driver.Blueprint) string {
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

func (r *Grammar) CompileDefault(_ driver.Blueprint, _ *driver.Command) string {
	return ""
}

func (r *Grammar) CompileDisableForeignKeyConstraints() string {
	return "SET FOREIGN_KEY_CHECKS=0;"
}

func (r *Grammar) CompileDrop(blueprint driver.Blueprint) string {
	return fmt.Sprintf("drop table %s", r.wrap.Table(blueprint.GetTableName()))
}

func (r *Grammar) CompileDropAllDomains(_ []string) string {
	return ""
}

func (r *Grammar) CompileDropAllTables(_ string, tables []driver.Table) []string {
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

func (r *Grammar) CompileDropAllTypes(_ string, _ []driver.Type) []string {
	return nil
}

func (r *Grammar) CompileDropAllViews(_ string, views []driver.View) []string {
	var dropViews []string
	for _, view := range views {
		dropViews = append(dropViews, view.Name)
	}

	return []string{
		fmt.Sprintf("drop view %s", r.wrap.Columnize(dropViews)),
	}
}

func (r *Grammar) CompileDropColumn(blueprint driver.Blueprint, command *driver.Command) []string {
	columns := r.wrap.PrefixArray("drop", r.wrap.Columns(command.Columns))

	return []string{
		fmt.Sprintf("alter table %s %s", r.wrap.Table(blueprint.GetTableName()), strings.Join(columns, ", ")),
	}
}

func (r *Grammar) CompileDropForeign(blueprint driver.Blueprint, command *driver.Command) string {
	return fmt.Sprintf("alter table %s drop foreign key %s", r.wrap.Table(blueprint.GetTableName()), r.wrap.Column(command.Index))
}

func (r *Grammar) CompileDropFullText(blueprint driver.Blueprint, command *driver.Command) string {
	return r.CompileDropIndex(blueprint, command)
}

func (r *Grammar) CompileDropIfExists(blueprint driver.Blueprint) string {
	return fmt.Sprintf("drop table if exists %s", r.wrap.Table(blueprint.GetTableName()))
}

func (r *Grammar) CompileDropIndex(blueprint driver.Blueprint, command *driver.Command) string {
	return fmt.Sprintf("alter table %s drop index %s", r.wrap.Table(blueprint.GetTableName()), r.wrap.Column(command.Index))
}

func (r *Grammar) CompileDropPrimary(blueprint driver.Blueprint, _ *driver.Command) string {
	return fmt.Sprintf("alter table %s drop primary key", r.wrap.Table(blueprint.GetTableName()))
}

func (r *Grammar) CompileDropUnique(blueprint driver.Blueprint, command *driver.Command) string {
	return r.CompileDropIndex(blueprint, command)
}

func (r *Grammar) CompileEnableForeignKeyConstraints() string {
	return "SET FOREIGN_KEY_CHECKS=1;"
}

func (r *Grammar) CompileForeign(blueprint driver.Blueprint, command *driver.Command) string {
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

func (r *Grammar) CompileFullText(blueprint driver.Blueprint, command *driver.Command) string {
	return r.compileKey(blueprint, command, "fulltext")
}

func (r *Grammar) CompileIndex(blueprint driver.Blueprint, command *driver.Command) string {
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

func (r *Grammar) CompileJsonContains(column string, value any, isNot bool) (string, []any, error) {
	field, path := r.wrap.JsonFieldAndPath(column)
	binding, err := App.GetJson().Marshal(value)
	if err != nil {
		return column, nil, err
	}

	return r.wrap.Not(fmt.Sprintf("json_contains(%s, ?%s)", field, path), isNot), []any{string(binding)}, nil
}

func (r *Grammar) CompileJsonContainsKey(column string, isNot bool) string {
	field, path := r.wrap.JsonFieldAndPath(column)

	return r.wrap.Not(fmt.Sprintf("ifnull(json_contains_path(%s, 'one'%s), 0)", field, path), isNot)
}

func (r *Grammar) CompileJsonLength(column string) string {
	field, path := r.wrap.JsonFieldAndPath(column)

	return fmt.Sprintf("json_length(%s%s)", field, path)
}

func (r *Grammar) CompileJsonSelector(column string) string {
	field, path := r.wrap.JsonFieldAndPath(column)

	return fmt.Sprintf("json_unquote(json_extract(%s%s))", field, path)
}

func (r *Grammar) CompileJsonValues(args ...any) []any {
	for i, arg := range args {
		val := reflect.ValueOf(arg)
		if val.Kind() == reflect.Ptr {
			if val.IsNil() {
				continue
			}
			val = val.Elem()
		}
		switch val.Kind() {
		case reflect.Bool:
			args[i] = fmt.Sprint(val.Interface())

		case reflect.Slice, reflect.Array:
			if length := val.Len(); length > 0 {
				values := make([]any, length)
				for j := 0; j < length; j++ {
					values[j] = val.Index(j).Interface()
				}
				args[i] = r.CompileJsonValues(values...)
			}
		default:

		}

	}
	return args
}

func (r *Grammar) CompileLockForUpdate(builder sq.SelectBuilder, conditions *driver.Conditions) sq.SelectBuilder {
	if conditions.LockForUpdate != nil && *conditions.LockForUpdate {
		builder = builder.Suffix("FOR UPDATE")
	}

	return builder
}

func (r *Grammar) CompileLockForUpdateForGorm() clause.Expression {
	return clause.Locking{Strength: "UPDATE"}
}

func (r *Grammar) CompilePlaceholderFormat() driver.PlaceholderFormat {
	return nil
}

func (r *Grammar) CompilePrimary(blueprint driver.Blueprint, command *driver.Command) string {
	var algorithm string
	if command.Algorithm != "" {
		algorithm = "using " + command.Algorithm
	}

	return fmt.Sprintf("alter table %s add primary key %s(%s)", r.wrap.Table(blueprint.GetTableName()), algorithm, r.wrap.Columnize(command.Columns))
}

func (r *Grammar) CompileInRandomOrder(builder sq.SelectBuilder, conditions *driver.Conditions) sq.SelectBuilder {
	if conditions.InRandomOrder != nil && *conditions.InRandomOrder {
		conditions.OrderBy = []string{"RAND()"}
	}

	return builder
}

func (r *Grammar) CompileRandomOrderForGorm() string {
	return "RAND()"
}

func (r *Grammar) CompileRename(blueprint driver.Blueprint, command *driver.Command) string {
	return fmt.Sprintf("rename table %s to %s", r.wrap.Table(blueprint.GetTableName()), r.wrap.Table(command.To))
}

func (r *Grammar) CompileRenameColumn(blueprint driver.Blueprint, command *driver.Command, columns []driver.Column) (string, error) {
	if v, err := semver.NewVersion(r.version); err == nil {
		isMariaDB := r.name != Name
		if (isMariaDB && v.LessThan(semver.New(10, 5, 2, "", ""))) || (!isMariaDB && v.LessThan(semver.New(8, 0, 3, "", ""))) {
			return r.compileLegacyRenameColumn(blueprint, command, columns)
		}
	}

	return fmt.Sprintf("alter table %s rename column %s to %s",
		r.wrap.Table(blueprint.GetTableName()),
		r.wrap.Column(command.From),
		r.wrap.Column(command.To),
	), nil
}

func (r *Grammar) CompileRenameIndex(blueprint driver.Blueprint, command *driver.Command, _ []driver.Index) []string {
	return []string{
		fmt.Sprintf("alter table %s rename index %s to %s", r.wrap.Table(blueprint.GetTableName()), r.wrap.Column(command.From), r.wrap.Column(command.To)),
	}
}

func (r *Grammar) CompileSharedLock(builder sq.SelectBuilder, conditions *driver.Conditions) sq.SelectBuilder {
	if conditions.SharedLock != nil && *conditions.SharedLock {
		builder = builder.Suffix("FOR SHARE")
	}

	return builder
}

func (r *Grammar) CompileSharedLockForGorm() clause.Expression {
	return clause.Locking{Strength: "SHARE"}
}

func (r *Grammar) CompileTables(database string) string {
	return fmt.Sprintf("select table_name as `name`, (data_length + index_length) as `size`, "+
		"table_comment as `comment`, engine as `engine`, table_collation as `collation` "+
		"from information_schema.tables where table_schema = %s and table_type in ('BASE TABLE', 'SYSTEM VERSIONED') "+
		"order by table_name", r.wrap.Quote(database))
}

func (r *Grammar) CompileTableComment(blueprint driver.Blueprint, command *driver.Command) string {
	return fmt.Sprintf("alter table %s comment = '%s'",
		r.wrap.Table(blueprint.GetTableName()),
		strings.ReplaceAll(command.Value, "'", "''"),
	)
}

func (r *Grammar) CompileTypes() string {
	return ""
}

func (r *Grammar) CompileUnique(blueprint driver.Blueprint, command *driver.Command) string {
	return r.compileKey(blueprint, command, "unique")
}

func (r *Grammar) CompileVersion() string {
	return "SELECT VERSION() AS value;"
}

func (r *Grammar) CompileViews(database string) string {
	return fmt.Sprintf("select table_name as `name`, view_definition as `definition` "+
		"from information_schema.views where table_schema = %s "+
		"order by table_name", r.wrap.Quote(database))
}

func (r *Grammar) GetAttributeCommands() []string {
	return r.attributeCommands
}

func (r *Grammar) ModifyAfter(_ driver.Blueprint, column driver.ColumnDefinition) string {
	if column.GetAfter() != "" {
		return fmt.Sprintf(" after %s", r.wrap.Column(column.GetAfter()))
	}

	return ""
}

func (r *Grammar) ModifyComment(_ driver.Blueprint, column driver.ColumnDefinition) string {
	if comment := column.GetComment(); comment != "" {
		// Escape special characters to prevent SQL injection
		comment = strings.ReplaceAll(comment, "'", "''")
		comment = strings.ReplaceAll(comment, "\\", "\\\\")

		return fmt.Sprintf(" comment '%s'", comment)
	}

	return ""
}

func (r *Grammar) ModifyDefault(_ driver.Blueprint, column driver.ColumnDefinition) string {
	if column.GetDefault() != nil {
		return fmt.Sprintf(" default %s", schema.ColumnDefaultValue(column.GetDefault()))
	}

	return ""
}

func (r *Grammar) ModifyNullable(_ driver.Blueprint, column driver.ColumnDefinition) string {
	if column.GetNullable() {
		return " null"
	} else {
		return " not null"
	}
}

func (r *Grammar) ModifyFirst(_ driver.Blueprint, column driver.ColumnDefinition) string {
	if column.IsFirst() {
		return " first"
	}

	return ""
}

func (r *Grammar) ModifyIncrement(blueprint driver.Blueprint, column driver.ColumnDefinition) string {
	if slices.Contains(r.serials, column.GetType()) && column.GetAutoIncrement() {
		if blueprint.HasCommand("primary") {
			return "auto_increment"
		}
		return " auto_increment primary key"
	}

	return ""
}

func (r *Grammar) ModifyOnUpdate(_ driver.Blueprint, column driver.ColumnDefinition) string {
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

func (r *Grammar) ModifyUnsigned(_ driver.Blueprint, column driver.ColumnDefinition) string {
	if column.GetUnsigned() {
		return " unsigned"
	}

	return ""
}

func (r *Grammar) TypeBigInteger(_ driver.ColumnDefinition) string {
	return "bigint"
}

func (r *Grammar) TypeBoolean(_ driver.ColumnDefinition) string {
	return "tinyint(1)"
}

func (r *Grammar) TypeChar(column driver.ColumnDefinition) string {
	return fmt.Sprintf("char(%d)", column.GetLength())
}

func (r *Grammar) TypeDate(_ driver.ColumnDefinition) string {
	return "date"
}

func (r *Grammar) TypeDateTime(column driver.ColumnDefinition) string {
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

func (r *Grammar) TypeDateTimeTz(column driver.ColumnDefinition) string {
	return r.TypeDateTime(column)
}

func (r *Grammar) TypeDecimal(column driver.ColumnDefinition) string {
	return fmt.Sprintf("decimal(%d, %d)", column.GetTotal(), column.GetPlaces())
}

func (r *Grammar) TypeDouble(_ driver.ColumnDefinition) string {
	return "double"
}

func (r *Grammar) TypeEnum(column driver.ColumnDefinition) string {
	return fmt.Sprintf(`enum(%s)`, strings.Join(r.wrap.Quotes(cast.ToStringSlice(column.GetAllowed())), ", "))
}

func (r *Grammar) TypeFloat(column driver.ColumnDefinition) string {
	precision := column.GetPrecision()
	if precision > 0 {
		return fmt.Sprintf("float(%d)", precision)
	}

	return "float"
}

func (r *Grammar) TypeInteger(_ driver.ColumnDefinition) string {
	return "int"
}

func (r *Grammar) TypeJson(_ driver.ColumnDefinition) string {
	return "json"
}

func (r *Grammar) TypeJsonb(_ driver.ColumnDefinition) string {
	return "json"
}

func (r *Grammar) TypeLongText(_ driver.ColumnDefinition) string {
	return "longtext"
}

func (r *Grammar) TypeMediumInteger(_ driver.ColumnDefinition) string {
	return "mediumint"
}

func (r *Grammar) TypeMediumText(_ driver.ColumnDefinition) string {
	return "mediumtext"
}

func (r *Grammar) TypeSmallInteger(_ driver.ColumnDefinition) string {
	return "smallint"
}

func (r *Grammar) TypeString(column driver.ColumnDefinition) string {
	length := column.GetLength()
	if length > 0 {
		return fmt.Sprintf("varchar(%d)", length)
	}

	return "varchar(255)"
}

func (r *Grammar) TypeText(_ driver.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeTime(column driver.ColumnDefinition) string {
	if column.GetPrecision() > 0 {
		return fmt.Sprintf("time(%d)", column.GetPrecision())
	} else {
		return "time"
	}
}

func (r *Grammar) TypeTimeTz(column driver.ColumnDefinition) string {
	return r.TypeTime(column)
}

func (r *Grammar) TypeTimestamp(column driver.ColumnDefinition) string {
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

func (r *Grammar) TypeTimestampTz(column driver.ColumnDefinition) string {
	return r.TypeTimestamp(column)
}

func (r *Grammar) TypeTinyInteger(_ driver.ColumnDefinition) string {
	return "tinyint"
}

func (r *Grammar) TypeTinyText(_ driver.ColumnDefinition) string {
	return "tinytext"
}

func (r *Grammar) addModifiers(sql string, blueprint driver.Blueprint, column driver.ColumnDefinition) string {
	for _, modifier := range r.modifiers {
		sql += modifier(blueprint, column)
	}

	return sql
}

func (r *Grammar) compileKey(blueprint driver.Blueprint, command *driver.Command, ttype string) string {
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

func (r *Grammar) compileLegacyRenameColumn(blueprint driver.Blueprint, command *driver.Command, columns []driver.Column) (string, error) {
	columns = collect.Filter(columns, func(c driver.Column, _ int) bool {
		return c.Name == command.From
	})
	if len(columns) == 0 {
		return "", errors.New(fmt.Sprintf("Column %s does not exist", command.From))
	}

	sql := columns[0].Type
	if len(columns[0].Collation) > 0 {
		sql += " collate " + columns[0].Collation
	}

	return fmt.Sprintf("alter table %s change %s %s %s",
		r.wrap.Table(blueprint.GetTableName()),
		r.wrap.Column(command.From),
		r.wrap.Column(command.To),
		r.addModifiers(sql, blueprint, r.rebuildColumnDefinition(columns[0])),
	), nil
}

func (r *Grammar) getColumns(blueprint driver.Blueprint) []string {
	var columns []string
	for _, column := range blueprint.GetAddedColumns() {
		columns = append(columns, r.getColumn(blueprint, column))
	}

	return columns
}

func (r *Grammar) getColumn(blueprint driver.Blueprint, column driver.ColumnDefinition) string {
	sql := fmt.Sprintf("%s %s", r.wrap.Column(column.GetName()), schema.ColumnType(r, column))

	return r.addModifiers(sql, blueprint, column)
}

func (r *Grammar) rebuildColumnDefinition(column driver.Column) driver.ColumnDefinition {
	definition := schema.NewColumnDefinition(column.Name, column.Type)
	if column.Autoincrement {
		definition.AutoIncrement()
	}
	if len(column.Comment) > 0 {
		definition.Comment(column.Comment)
	}
	if len(column.Default) > 0 {
		definition.Default(schema.Expression(column.Default))
	}
	if column.Nullable {
		definition.Nullable()
	}
	if len(column.Extra) > 0 && strings.HasPrefix(strings.ToLower(column.Extra), "on update") {
		onUpdate := strings.TrimPrefix(strings.TrimPrefix(column.Extra, "on update"), "ON UPDATE")
		definition.OnUpdate(schema.Expression(onUpdate))
	}

	return definition.Change()
}

func getCommandByName(commands []*driver.Command, name string) *driver.Command {
	commands = getCommandsByName(commands, name)
	if len(commands) == 0 {
		return nil
	}

	return commands[0]
}

func getCommandsByName(commands []*driver.Command, name string) []*driver.Command {
	var filteredCommands []*driver.Command
	for _, command := range commands {
		if command.Name == name {
			filteredCommands = append(filteredCommands, command)
		}
	}

	return filteredCommands
}
