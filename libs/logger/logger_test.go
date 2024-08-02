package logger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gitlab.com/telkom/monitoring-app/libs/logger"
)

type Suite struct {
	suite.Suite
}

func (c *Suite) SetupSuite() {}

func (c *Suite) TestLogger() {
	log := logger.New()

	c.Run("Get must be not nil", func() {
		assert.NotNil(c.T(), log.Get())
	})

	c.Run("Set for system log", func() {
		assert.NotNil(c.T(), log.UseForSystemLog())
	})

	c.Run("Print", func() {

		c.Run("Init json", func() {
			assert.NotNil(c.T(), log.Init(logger.Config{
				Level:  "debug",
				Format: "json",
			}))

			c.Run("json", func() {
				log.Get().Info("test json")
			})

		})

		c.Run("Init text", func() {
			assert.NotNil(c.T(), log.Init(logger.Config{
				Level:  "debug",
				Format: "text",
			}))

			c.Run("text", func() {
				log.Get().Info("test text")
			})

		})

	})

}

func TestLogger(t *testing.T) {
	suite.Run(t, &Suite{})
}
