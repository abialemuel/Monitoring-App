module gitlab.com/telkom/monitoring-app

go 1.22.0

require (
	github.com/prometheus/client_golang v1.19.1
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/viper v1.18.2
	github.com/stretchr/testify v1.9.0
	gitlab.playcourt.id/telkom-digital/dpe/modules/tlkm v0.0.27
	gitlab.playcourt.id/telkom-digital/dpe/std/impl/netmonk/Proto v0.0.7
	gitlab.playcourt.id/telkom-digital/dpe/std/impl/netmonk/prometheus-exporter v0.0.9
	go.opentelemetry.io/otel v1.19.0
	go.uber.org/mock v0.4.0
	google.golang.org/protobuf v1.34.1
)