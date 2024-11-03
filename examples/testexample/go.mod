module github.com/hpc-gridware/go-clusterscheduler/examples/testexample

go 1.23.1

replace github.com/hpc-gridware/go-clusterscheduler => ../..

require (
	github.com/hpc-gridware/go-clusterscheduler v0.0.0-20241027163340-55dac298d370
	go.uber.org/zap v1.27.0
	google.golang.org/protobuf v1.35.1
)

require (
	github.com/goccy/go-json v0.10.3 // indirect
	go.uber.org/multierr v1.10.0 // indirect
)
