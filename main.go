package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	mainCfg "gitlab.com/telkom/monitoring-app/config"
	"gitlab.com/telkom/monitoring-app/libs/logger"
	"gitlab.playcourt.id/telkom-digital/dpe/modules/tlkm/infrastructure/apm"
	"gitlab.playcourt.id/telkom-digital/dpe/std/impl/netmonk/prometheus-exporter/blackbox"
	"gitlab.playcourt.id/telkom-digital/dpe/std/impl/netmonk/prometheus-exporter/snmp"
)

var (
	blackboxProbe blackbox.Blackbox
	snmpProbe     snmp.Snmp
	err           error
	APM           *apm.APM
)

func main() {
	// config
	cfg := initializeConfig("config.yaml")

	log := initializeLogger(cfg)

	// initialize apm
	if cfg.Get().APM.Enabled {
		// using tlkm apm wrapper
		host := fmt.Sprintf("%s:%d", cfg.Get().APM.Host, cfg.Get().APM.Port)
		apmPayload := apm.APMPayload{
			ServiceHost:    &host,
			ServiceName:    cfg.Get().App.Name,
			ServiceEnv:     cfg.Get().App.Env,
			ServiceTribe:   cfg.Get().App.Tribe,
			ServiceVersion: cfg.Get().App.Version,
			SampleRate:     cfg.Get().APM.Rate,
		}
		APM, err = apm.NewAPM(apm.DatadogAPMType, apmPayload)
		if err != nil {
			log.Get().Error(err)
			panic(err)
		}
		defer APM.EndAPM()
	}

	moduleLogLevel := cfg.Get().Log.Level
	if moduleLogLevel == "fatal" {
		moduleLogLevel = "error"
	}
	// initialize blackbox
	blackboxProbe = initializeBlackbox(cfg.Get().Probe.NormalTimeout, moduleLogLevel)

	// initialize snmp
	snmpProbe = initializeSnmp(cfg.Get().Probe.NormalTimeout, moduleLogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to listen for interrupt signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// run primary consumer

	// Wait for an interrupt signal
	select {
	case <-interrupt:
		fmt.Println("Interrupt signal received")
	case <-ctx.Done():
		fmt.Println("Context done")
	}

	fmt.Println("Starting graceful shutdown")

	fmt.Println("Graceful shutdown complete")
}

func initializeLogger(cfg mainCfg.Config) logger.Logger {
	fmt.Println("Global Probe started...")
	log := logger.New().Init(logger.Config{
		Level:  cfg.Get().Log.Level,
		Format: cfg.Get().Log.Format,
	})
	return log
}

func initializeConfig(path string) mainCfg.Config {
	cfg := mainCfg.New()
	err := cfg.Init(path)
	if err != nil {
		fmt.Errorf("failed to load config: %s", err.Error())
		panic(err)
	}
	return cfg
}

func initializeBlackbox(timeout float64, logLevel string) blackbox.Blackbox {
	blackboxProbe, err := blackbox.New(0, timeout, logLevel)
	if err != nil {
		panic(err)
	}
	return blackboxProbe
}

func initializeSnmp(timeout float64, logLevel string) snmp.Snmp {
	snmpProbe, err := snmp.New(0, timeout, logLevel)
	if err != nil {
		panic(err)
	}
	return snmpProbe
}
