package main

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	mainCfg "gitlab.com/telkom/monitoring-app/config"
)

func TestInitializeConfig(t *testing.T) {
	cfg := initializeConfig("config.yaml")
	assert.NotNil(t, cfg)
}

func TestInitializeLogger(t *testing.T) {
	cfg := initializeConfig("config.yaml")

	log = initializeLogger(cfg)
	assert.NotNil(t, log)
}

func TestStartProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	probe := mainCfg.WorkerProbe{
		Ip:       "http://example.com",
		Interval: 1,
		ProbeConfig: &mainCfg.WebsiteConfig{
			Method: "GET",
		},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startProbe(ctx, probe)
	}()

	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()
}

func TestExecuteProbe(t *testing.T) {
	cfg := initializeConfig("config.yaml")

	log = initializeLogger(cfg)
	ctx := context.Background()
	probe := mainCfg.WorkerProbe{
		Ip:       "http://example.com",
		Interval: 1,
		ProbeConfig: &mainCfg.WebsiteConfig{
			Method: "GET",
		},
	}

	executeProbe(ctx, probe)
}

func TestMain(m *testing.M) {
	// Setup code here

	// Run tests
	code := m.Run()

	// Teardown code here

	os.Exit(code)
}
