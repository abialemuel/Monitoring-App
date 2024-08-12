package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/telkom/monitoring-app/config"
	mainCfg "gitlab.com/telkom/monitoring-app/config"
	"gitlab.com/telkom/monitoring-app/helper"
	"gitlab.com/telkom/monitoring-app/libs/logger"
	"gitlab.playcourt.id/telkom-digital/dpe/modules/tlkm/infrastructure/apm"
	proto "gitlab.playcourt.id/telkom-digital/dpe/std/impl/netmonk/Proto/interfaces"
	"go.opentelemetry.io/otel/attribute"
	"gopkg.in/yaml.v2"
)

var (
	APM          *apm.APM
	log          logger.Logger
	probeResults sync.Map
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

	var body string

	// Resolve dependencies dynamically
	const maxRetries = 3
	const retryInterval = time.Second * 3

	for _, dependency := range probe.Dependencies {
		// Retry loop for each dependency
		for retry := 0; retry < maxRetries; retry++ {
			if result, ok := probeResults.Load(dependency); ok {
				// Replace placeholders in the body with the result
				if probe.ProbeConfig.Body != "" {
					body = helper.ReplacePlaceholders(probe.ProbeConfig.Body, dependency, result.(string))
					probe.ProbeConfig.Body = body
				}

				// also replace placeholders in headers which key is Authorization
				for key, value := range probe.ProbeConfig.Headers {
					if key == "Authorization" {
						probe.ProbeConfig.Headers[key] = helper.ReplacePlaceholders(value, dependency, result.(string))
					}
				}

				// also replace placeholders in query
				for key, value := range probe.ProbeConfig.Query {
					probe.ProbeConfig.Query[key] = helper.ReplacePlaceholders(value, dependency, result.(string))
				}

				break // Dependency resolved, move to next dependency
			} else {
				log.Get().WithFields(logrus.Fields{
					"ip":        probe.Ip,
					"interval":  probe.Interval,
					"tribe":     probe.Tribe,
					"operation": probe.Operation,
				}).Warnf("Waiting for dependency %s (retry %d/%d)", dependency, retry+1, maxRetries)
				time.Sleep(retryInterval) // Wait before retrying
			}
		}
		// If the dependency is not resolved after maxRetries, log an error
		if _, ok := probeResults.Load(dependency); !ok {
			log.Get().WithFields(logrus.Fields{
				"ip":        probe.Ip,
				"interval":  probe.Interval,
				"tribe":     probe.Tribe,
				"operation": probe.Operation,
			}).Errorf("Dependency %s not resolved after %d retries", dependency, maxRetries)
			return
		}
	}

	req, err := http.NewRequest(probe.ProbeConfig.Method, probe.Ip, bytes.NewBuffer([]byte(probe.ProbeConfig.Body)))
	if err != nil {
		log.Get().WithFields(logrus.Fields{
			"ip":        probe.Ip,
			"interval":  probe.Interval,
			"tribe":     probe.Tribe,
			"operation": probe.Operation,
		}).Error(err)
		return
	}

	for key, value := range probe.ProbeConfig.Headers {
		req.Header.Set(key, value)
	}
	if auth.Username != "" && auth.Password != "" {
		req.SetBasicAuth(auth.Username, auth.Password)
	}

	if probe.ProbeConfig.Query != nil {
		q := req.URL.Query()
		for key, value := range probe.ProbeConfig.Query {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	ctx, span := apm.StartTransaction(ctx, fmt.Sprintf("%s.%s", probe.Tribe, probe.Operation))
	apm.AddEvent(ctx, "ProbeDevice",
		attribute.String("ip", probe.Ip),
		attribute.String("operation", probe.Operation),
		attribute.String("tribe", probe.Tribe),
		attribute.Int("interval", probe.Interval),
	)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Get().WithFields(logrus.Fields{
			"trace_id":  apm.GetTraceID(ctx),
			"ip":        probe.Ip,
			"interval":  probe.Interval,
			"tribe":     probe.Tribe,
			"operation": probe.Operation,
		}).Error(err)
		return
	}
	// all status code with 2xx is considered as success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Get().WithFields(logrus.Fields{
			"trace_id":  apm.GetTraceID(ctx),
			"ip":        probe.Ip,
			"interval":  probe.Interval,
			"tribe":     probe.Tribe,
			"operation": probe.Operation,
			"status":    resp.Status,
		}).Error("ProbeFailed")

		apm.AddEvent(ctx, "ProbeFailed",
			attribute.String("ip", probe.Ip),
			attribute.String("error", resp.Status),
		)
	}

	apm.EndTransaction(span)

	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Get().WithFields(logrus.Fields{
			"trace_id": apm.GetTraceID(ctx),
			"ip":       probe.Ip,
			"interval": probe.Interval,
		}).Error(err)
		return
	}

	// Store response.body in probeResults
	// to be used as a dependency in the next probe
	probeResults.Store(probe.Operation, string(bodyBytes))

	log.Get().WithFields(logrus.Fields{
		"trace_id":  apm.GetTraceID(ctx),
		"ip":        probe.Ip,
		"interval":  probe.Interval,
		"tribe":     probe.Tribe,
		"operation": probe.Operation,
		"query":     req.URL.RawQuery,
	}).Info("ProbeFinished")
}
