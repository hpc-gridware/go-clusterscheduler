# Open Cluster Scheduler Go API

Welcome to the **Open Cluster Scheduler Go API** repository. This project provides
a Go-based API to interact with the Open Cluster Scheduler, encapsulating the
command-line interface to streamline the configuration and management of
Open/Gridware Cluster Scheduler clusters.

## Features

The Go API offers a range of powerful features, including:

- **`qconf` Command Line Wrapper**: This primary feature enables
developers to build robust applications to configure the Open/Gridware
Cluster Scheduler effortlessly.
- **`simulator` Application**: This application implements a
container based simulator of a real "Grid Engine" cluster. It can
dump the configuration of a SGE or Open Grid Engine cluster and
apply changes to a cluster simulated within a container.
(See the [`cmd/simulator`](cmd/simulator) directory for more information.)
- **`sharetree` GUI Editor**: This application implements a
simple web-based visualization and editing tool for Open Cluster
Scheduler (OCS) and Gridware Cluster Scheduler (GCS) sharetree
configurations.
(See the [`cmd/sharetree`](cmd/sharetree) directory for more information.)
- **[mcp-server](https://github.com/hpc-gridware/go-clusterscheduler/tree/main/cmd/clusterscheduler-mcp)**: Implements an example MCP (Model Context Protocol) for
interacting with the cluster. Process job details, accounting information,
cluster configuration details with your favorite AI application (like Claude)
which supports MCP extensions.

## Go API Development Container

This project includes scripts to set up a one-node Open Cluster Scheduler
cluster, which helps you to quickly build and test Go API applications
without manually constructing a build environment.

Note, that the tests are written for this container setup. You should
not run the tests against a real cluster as they will modify the cluster
configuration of course.

## Getting Started

### Prerequisites

To begin, ensure you have the following software installed:

- Docker
- `make` tool

### Building and Testing Using the Container

#### Build the Container

First, build the container, which is based on Ubuntu 22.04
and includes all required dependencies:

```bash
make build
```

#### Run the Single Node Cluster

After successfully building the container, you can initialize and
run a single-node cluster:

```bash
make run
```

Upon the first successful execution of `make run`, you can expect
command line output similar to the following:

```shell
Install log can be found in: /opt/cs-install/default/common/install_logs/qmaster_install_master_2024-08-11_12:00:00.log
Install log can be found in: /opt/cs-install/default/common/install_logs/execd_install_master_2024-08-11_12:00:00.log
root@master modified "global" in configuration list
root@master modified "all.q" in cluster queue list
```

This indicates that the cluster has been successfully set up. You can
check the status of the cluster by running the following command:

```bash
> qhost
HOSTNAME                ARCH         NCPU NSOC NCOR NTHR  LOAD  MEMTOT  MEMUSE  SWAPTO  SWAPUS
----------------------------------------------------------------------------------------------
global                  -               -    -    -    -     -       -       -       -       -
master                  lx-amd64        7    1    7    7  0.37   23.5G  838.0M    3.0G     0.0
> cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler/pkg/qconf
> ginkgo -v
...
```

The installation directly is the local "installation" subdirectory on the
the host which is mounted into the container. The subsequent runs of `make run`
will reuse the existing installation for faster startup. If you want to
reinstall the cluster, you can remove the `installation` directory and
run `make run` again.

## Issues

If you encounter any issues or have questions, please open an issue in
this repository. We'll be happy to assist.

- No jobs are scheduled: When using a laptop and closing and re-opening, the
  scheduler thread times out. To protect the qmaster, the scheduler thread
  gets disconnect. To get it running again either restart the container or
  remove and attach the scheduler thread manually ("qconf -kt scheduler" and
  then "qconf -at scheduler").
