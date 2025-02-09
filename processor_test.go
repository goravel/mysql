package mysql

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/goravel/framework/contracts/database/schema"
)

type ProcessorTestSuite struct {
	suite.Suite
	processor *Processor
}

func TestProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessorTestSuite))
}

func (s *ProcessorTestSuite) SetupTest() {
	s.processor = NewProcessor()
}

func (s *ProcessorTestSuite) TestProcessColumns() {
	tests := []struct {
		name      string
		dbColumns []schema.DBColumn
		expected  []schema.Column
	}{
		{
			name: "ValidInput",
			dbColumns: []schema.DBColumn{
				{Name: "id", Type: "int", TypeName: "INT", Nullable: "NO", Extra: "auto_increment", Collation: "utf8_general_ci", Comment: "primary key", Default: "0"},
				{Name: "name", Type: "varchar", TypeName: "VARCHAR", Nullable: "YES", Extra: "", Collation: "utf8_general_ci", Comment: "user name", Default: ""},
			},
			expected: []schema.Column{
				{Autoincrement: true, Collation: "utf8_general_ci", Comment: "primary key", Default: "0", Extra: "auto_increment", Name: "id", Nullable: false, Type: "int", TypeName: "INT"},
				{Autoincrement: false, Collation: "utf8_general_ci", Comment: "user name", Default: "", Name: "name", Nullable: true, Type: "varchar", TypeName: "VARCHAR"},
			},
		},
		{
			name:      "EmptyInput",
			dbColumns: []schema.DBColumn{},
		},
		{
			name: "NullableColumn",
			dbColumns: []schema.DBColumn{
				{Name: "description", Type: "text", TypeName: "TEXT", Nullable: "YES", Extra: "", Collation: "utf8_general_ci", Comment: "description", Default: ""},
			},
			expected: []schema.Column{
				{Autoincrement: false, Collation: "utf8_general_ci", Comment: "description", Default: "", Name: "description", Nullable: true, Type: "text", TypeName: "TEXT"},
			},
		},
		{
			name: "NonNullableColumn",
			dbColumns: []schema.DBColumn{
				{Name: "created_at", Type: "timestamp", TypeName: "TIMESTAMP", Nullable: "NO", Extra: "", Collation: "", Comment: "creation time", Default: "CURRENT_TIMESTAMP"},
			},
			expected: []schema.Column{
				{Autoincrement: false, Collation: "", Comment: "creation time", Default: "CURRENT_TIMESTAMP", Name: "created_at", Nullable: false, Type: "timestamp", TypeName: "TIMESTAMP"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := s.processor.ProcessColumns(tt.dbColumns)
			s.Equal(tt.expected, result)
		})
	}
}

func (s *ProcessorTestSuite) TestProcessForeignKeys() {
	tests := []struct {
		name          string
		dbForeignKeys []schema.DBForeignKey
		expected      []schema.ForeignKey
	}{
		{
			name: "ValidInput",
			dbForeignKeys: []schema.DBForeignKey{
				{Name: "fk_user_id", Columns: "user_id", ForeignSchema: "public", ForeignTable: "users", ForeignColumns: "id", OnUpdate: "CASCADE", OnDelete: "SET NULL"},
			},
			expected: []schema.ForeignKey{
				{Name: "fk_user_id", Columns: []string{"user_id"}, ForeignSchema: "public", ForeignTable: "users", ForeignColumns: []string{"id"}, OnUpdate: "cascade", OnDelete: "set null"},
			},
		},
		{
			name:          "EmptyInput",
			dbForeignKeys: []schema.DBForeignKey{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := s.processor.ProcessForeignKeys(tt.dbForeignKeys)
			s.Equal(tt.expected, result)
		})
	}
}

func (s *ProcessorTestSuite) TestProcessIndexes() {
	tests := []struct {
		name      string
		dbIndexes []schema.DBIndex
		expected  []schema.Index
	}{
		{
			name: "ValidInput",
			dbIndexes: []schema.DBIndex{
				{Name: "users_email_unique", Columns: "email", Type: "BTREE", Primary: false, Unique: true},
				{Name: "PRIMARY", Columns: "id", Type: "BTREE", Primary: true, Unique: true},
				{Name: "users_name_index", Columns: "first_name,last_name", Type: "BTREE", Primary: false, Unique: false},
			},
			expected: []schema.Index{
				{Name: "users_email_unique", Columns: []string{"email"}, Type: "btree", Primary: false, Unique: true},
				{Name: "primary", Columns: []string{"id"}, Type: "btree", Primary: true, Unique: true},
				{Name: "users_name_index", Columns: []string{"first_name", "last_name"}, Type: "btree", Primary: false, Unique: false},
			},
		},
		{
			name:      "EmptyInput",
			dbIndexes: []schema.DBIndex{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := s.processor.ProcessIndexes(tt.dbIndexes)
			s.Equal(tt.expected, result)
		})
	}
}
