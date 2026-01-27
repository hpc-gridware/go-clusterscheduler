module github.com/hpc-gridware/go-clusterscheduler/cmd/simulator

go 1.24.0

toolchain go1.24.11

replace github.com/hpc-gridware/go-clusterscheduler => ../..

require (
	github.com/hpc-gridware/go-clusterscheduler v0.0.0-20240826155740-e0d47e7b5d2d
	github.com/spf13/cobra v1.8.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)
