module github.com/hpc-gridware/go-clusterscheduler/cmd/adapter

go 1.23.1

replace github.com/hpc-gridware/go-clusterscheduler => ./../../

require (
	github.com/gorilla/mux v1.8.1
	github.com/hpc-gridware/go-clusterscheduler v0.0.0-20240914092021-0285da16cc31
	github.com/spf13/cobra v1.9.1
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelslog v0.8.0 // indirect
	go.opentelemetry.io/otel v1.33.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.9.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.33.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.33.0 // indirect
	go.opentelemetry.io/otel/log v0.9.0 // indirect
	go.opentelemetry.io/otel/metric v1.33.0 // indirect
	go.opentelemetry.io/otel/sdk v1.33.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.9.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.33.0 // indirect
	go.opentelemetry.io/otel/trace v1.33.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
)
