module github.com/hpc-gridware/go-clusterscheduler/pkg/adapter

go 1.23.1

require (
	github.com/gorilla/mux v1.8.1
	github.com/hpc-gridware/go-clusterscheduler v0.0.0-20240914052507-a59463d8ccd2
	go.opentelemetry.io/contrib/bridges/otelslog v0.5.0
	go.opentelemetry.io/otel v1.30.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.6.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.30.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.30.0
	go.opentelemetry.io/otel/log v0.6.0
	go.opentelemetry.io/otel/metric v1.30.0
	go.opentelemetry.io/otel/sdk v1.30.0
	go.opentelemetry.io/otel/sdk/log v0.6.0
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.30.0 // indirect
	go.opentelemetry.io/otel/trace v1.30.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
)
