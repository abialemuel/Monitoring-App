package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/telkom/monitoring-app/config"
	mainCfg "gitlab.com/telkom/monitoring-app/config"
	"gitlab.com/telkom/monitoring-app/libs/logger"
	"gitlab.playcourt.id/telkom-digital/dpe/modules/tlkm/infrastructure/apm"
	proto "gitlab.playcourt.id/telkom-digital/dpe/std/impl/netmonk/Proto/interfaces"
	"gitlab.playcourt.id/telkom-digital/dpe/std/impl/netmonk/prometheus-exporter/blackbox"
	"go.opentelemetry.io/otel/attribute"
	"gopkg.in/yaml.v2"
)

var (
	blackboxProbe  blackbox.Blackbox
	err            error
	APM            *apm.APM
	defaultModules = []*proto.Module{
		{
			Name: "blackbox",
			Config: map[string]string{
				"moduleName": "http_2xx",
			},
		},
	}
	log logger.Logger
)

func main() {
	// config
	cfg := initializeConfig("config.yaml")

	log = initializeLogger(cfg)

	// Load probes configuration from YAML file
	fileData, err := ioutil.ReadFile("probe_config.yaml")
	if err != nil {
		log.Get().Error(err)
		panic(err)
	}

	var probesConfig config.ProbesConfig
	err = yaml.Unmarshal(fileData, &probesConfig)
	if err != nil {
		log.Get().Error(err)
		panic(err)
	}

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
		fmt.Println("APM started...")
		defer APM.EndAPM()
	}

	moduleLogLevel := cfg.Get().Log.Level
	if moduleLogLevel == "fatal" {
		moduleLogLevel = "error"
	}
	// initialize blackbox
	blackboxProbe = initializeBlackbox(cfg.Get().Probe.NormalTimeout, moduleLogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to listen for interrupt signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Create and start probes
	for _, probe := range probesConfig.Probes {
		go startProbe(ctx, probe)
	}

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
	fmt.Println("Monitoring-app started...")
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

func startProbe(ctx context.Context, probe mainCfg.WorkerProbe) {
	ticker := time.NewTicker(time.Duration(probe.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go executeProbe(ctx, probe)
		case <-ctx.Done():
			return
		}
	}
}

func executeProbe(ctx context.Context, probe mainCfg.WorkerProbe) {
	ctx, span := apm.StartTransaction(ctx, "WorkerProbe.Serve")
	defer apm.EndTransaction(span)
	apm.AddEvent(ctx, "ProbeDevice",
		attribute.String("ip", probe.Ip),
		attribute.Int("interval", probe.Interval),
	)
	log.Get().WithFields(logrus.Fields{
		"trace_id": apm.GetTraceID(ctx),
		"ip":       probe.Ip,
		"interval": probe.Interval,
	}).Info("ProbeStarted")

	auth := &proto.Authorization{}
	if probe.ProbeConfig.Authorization != nil {
		auth = &proto.Authorization{
			Username: probe.ProbeConfig.Authorization.Username,
			Password: probe.ProbeConfig.Authorization.Password,
		}
	}
	workerProbe := proto.WorkerProbe{
		Ip:       probe.Ip,
		Interval: int32(probe.Interval),
		Modules:  defaultModules,
		ProbeConfig: &proto.WorkerProbe_Website{
			Website: &proto.WebsiteConfig{
				Method:        probe.ProbeConfig.Method,
				Headers:       probe.ProbeConfig.Headers,
				Authorization: auth,
			},
		},
	}

	res, err := blackboxProbe.Call(probe.Ip, defaultModules[0].Config["moduleName"], &workerProbe)
	if err != nil || !res.Success() {
		if err == nil {
			err = fmt.Errorf("probe metrics failed")
		}

		log.Get().WithFields(logrus.Fields{
			"trace_id": apm.GetTraceID(ctx),
			"ip":       probe.Ip,
			"interval": probe.Interval,
		}).Error(err)

		apm.AddEvent(ctx, "ProbeDeviceErr",
			attribute.String("error", err.Error()),
		)
		return
	}
	log.Get().WithFields(logrus.Fields{
		"trace_id": apm.GetTraceID(ctx),
		"ip":       probe.Ip,
		"interval": probe.Interval,
	}).Info("ProbeFinished")
}
