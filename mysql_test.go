package mysql

import (
	"testing"

	"github.com/goravel/framework/process"
	"github.com/goravel/framework/testing/utils"
	"github.com/goravel/mysql/contracts"
	mocks "github.com/goravel/mysql/mocks"
	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	t.Parallel()
	writes := []contracts.FullConfig{
		{
			Config: contracts.Config{
				Host:     "localhost",
				Database: "goravel",
				Username: "goravel",
				Password: "Framework!123",
			},
			Loc:     "UTC",
			Charset: "utf8mb4",
		},
	}

	docker := NewDocker(nil, process.New(), writes[0].Database, writes[0].Username, writes[0].Password)
	assert.NoError(t, docker.Build())

	writes[0].Port = docker.databaseConfig.Port
	_, err := docker.connect()
	assert.NoError(t, err)

	mockConfig := mocks.NewConfigBuilder(t)
	mockConfig.EXPECT().Writers().Return(writes).Once()
	mockConfig.EXPECT().Readers().Return([]contracts.FullConfig{}).Once()

	mysql := &Mysql{
		config: mockConfig,
		log:    utils.NewTestLog(),
	}
	version := mysql.getVersion()
	assert.Contains(t, version, ".")
	assert.NoError(t, docker.Shutdown())
}

// https://github.com/goravel/goravel/issues/706
func TestIssue706(t *testing.T) {
	t.Parallel()
	writes := []contracts.FullConfig{
		{
			Config: contracts.Config{
				Host:     "localhost",
				Database: "goravel",
				Username: "goravel",
				Password: "Framework!123",
			},
			Loc:     "Asia/Shanghai",
			Charset: "utf8mb4",
		},
	}

	docker := NewDocker(nil, process.New(), writes[0].Database, writes[0].Username, writes[0].Password)
	assert.NoError(t, docker.Build())

	writes[0].Port = docker.databaseConfig.Port
	_, err := docker.connect()
	assert.NoError(t, err)

	mockConfig := mocks.NewConfigBuilder(t)
	mockConfig.EXPECT().Writers().Return(writes).Once()
	mockConfig.EXPECT().Readers().Return([]contracts.FullConfig{}).Once()

	mysql := &Mysql{
		config: mockConfig,
		log:    utils.NewTestLog(),
	}
	version := mysql.getVersion()
	assert.Contains(t, version, ".")
	assert.NoError(t, docker.Shutdown())
}
