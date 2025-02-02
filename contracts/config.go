package contracts

import (
	contractsconfig "github.com/goravel/framework/contracts/config"
)

type ConfigBuilder interface {
	Config() contractsconfig.Config
	Connection() string
	Reads() []FullConfig
	Writes() []FullConfig
}

// Replacer replacer interface like strings.Replacer
type Replacer interface {
	Replace(name string) string
}

// Config Used in config/database.go
type Config struct {
	Dsn      string
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

// FullConfig Fill the default value for Config
type FullConfig struct {
	Config
	Charset      string
	Connection   string
	Driver       string
	Loc          string
	NameReplacer Replacer
	NoLowerCase  bool
	Prefix       string
	Singular     bool
}
