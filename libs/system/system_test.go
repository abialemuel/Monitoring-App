package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/abialemuel/monitoring-app/libs/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
}

func (c *Suite) SetupSuite() {}

func (c *Suite) TestSystem() {
	c.Run("New", func() {
		assert.NotNil(c.T(), system.New())
	})

	c.Run("Register", func() {
		s := system.New()
		assert.NotNil(c.T(), s.Register(func() {}))
	})

	c.Run("Run", func() {
		s := system.New()
		c.Run("mainFn returns nil", func() {
			assert.NotNil(c.T(), s.Run(func() error { return nil }))
		})
		c.Run("mainFn returns error", func() {
			assert.NotNil(c.T(), s.Run(func() error { return assert.AnError }))
		})
	})

	c.Run("Wait", func() {
		s := system.New()
		c.Run("terminated by mainChan", func() {
			s.Run(func() error {
				time.Sleep(100 * time.Millisecond)
				return assert.AnError
			}).Wait(func() {}, func(ctx context.Context) {}, func() error { return nil })
		})

		c.Run("terminated by Interrupt signal", func() {
			s.Run(func() error {
				time.Sleep(100 * time.Millisecond)
				s.Close()
				return nil
			}).Wait(func() error {
				return assert.AnError
			}, func(ctx context.Context) error {
				return assert.AnError
			})
		})
	})

	c.Run("Close", func() {
		s := system.New()
		assert.NotPanics(c.T(), func() { s.Close() })
	})

}

func TestConfig(t *testing.T) {
	suite.Run(t, &Suite{})
}
