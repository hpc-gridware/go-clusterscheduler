# Simulate a Grid Engine Cluster in a Open Cluster Scheduler Container

Do you want to simulate your Grid Engine or Open Cluster Scheduler?

Benefits:
- Testing new configurations without affecting your production cluster.
- Testing new versions of Open Cluster Scheduler with your existing configuration.
- Testing scheduler strategies and policies, JSV scripts, and more.
- Note, that jobs are simulated as well and never executed. The first
  parameter of the job defines the runtime.

This is the right place for you.

This repository contains a Dockerfile that builds a container with
Open Cluster Scheduler (OCS). Using the simulator application, you can
dump the configuration of your existing Grid Engine cluster and load
it into the Open Cluster Scheduler inside this container.

This simulator application allows you to do 2 things:
* Dump the configuration of your Grid Engine cluster into JSON format.
* Load the JSON configuration into the cluster inside the container.

Note, depending on the version of your Grid Engine cluster, it
might or might not work. Resons for configuration dumping might
not work: Lists which have different separation characters. We
need to catch all of them. Resons for configuration loading might
not work: The configuration file is not complete or has new mandatory
fields. We need to catch all of them.

Please report and issues you find to get the dump function it stable
accross different distributions and Grid Engine implementations.

## Build the simulation Go Application

To build the simulation application, you need to have Go installed
in your system. You can download Go from the [official website](https://golang.org/dl/).

Once you have Go installed, you can build the application by running
the following command:

```bash
go build
```

This will generate an executable file called `simulator` for your
target architecture.

## Dump the configuration of your Grid Engine / OCS / GCS cluster

To retrieve the configuration of your cluster you need to have
read access to the cluster. Check if e.g. *qconf -sconf* works in your
system. Depending of the amount of configuration objects (queues,
hosts, ...), this might take a while as there is a delay between each
executed *qconf* show command. The delay helps to be gentle with the
cluster.

```bash
./simulator dump > cluster.json
```

If that worked, you should have a file called `cluster.json` with
the configuration of your cluster. Important is to use the same
version of the simulator for dumping and loading the configuration.

## Run the Simulated Cluster

Go to the root of this repository and ensure that the *installation*
subdirectory is removed. If not removed, remove it.

Then run the following command:

```bash
make simulate
```

This will start the Open Cluster Scheduler container with the
configuration of your cluster inside *cluster.json*. For testing
puposes, there is already a *cluster.json* file in the repository.

When successful, you should see the following message:

```
Install log can be found in: /opt/cs-install/default/common/install_logs/qmaster_install_master_2024-09-02_07:57:33.log
Install log can be found in: /opt/cs-install/default/common/install_logs/execd_install_master_2024-09-02_07:57:50.log
root@master modified "global" in configuration list
root@master modified "all.q" in cluster queue list
go: downloading go1.22.5 (linux/amd64)
go: downloading github.com/spf13/cobra v1.8.1
go: downloading github.com/spf13/pflag v1.0.5
Hosts added to /etc/hosts
file: /tmp/load_report_host2170379353/load_report_host
Complex added to current configuration
Global configuration added to current configuration
Simulated cluster configuration applied
Restarting qmaster
waiting for qmaster to shut down...
   starting sge_qmaster
qmaster restarted
root@master:~/go/src/github.com/hpc-gridware/go-clusterscheduler/cmd/simulator# qhost
HOSTNAME                ARCH         NCPU NSOC NCOR NTHR  LOAD  MEMTOT  MEMUSE  SWAPTO  SWAPUS
----------------------------------------------------------------------------------------------
global                  -               -    -    -    -     -       -       -       -       -
master                  lx-amd64        7    1    7    7  0.79   23.5G  692.3M    3.0G     0.0
sim1                    lx-amd64        7    1    7    7  0.79   23.5G  692.3M    3.0G     0.0
sim2                    lx-amd64        7    1    7    7  0.79   23.5G  692.3M    3.0G     0.0
sim3                    lx-amd64        7    1    7    7  0.79   23.5G  692.3M    3.0G     0.0
sim4                    lx-amd64        7    1    7    7  0.79   23.5G  692.3M    3.0G     0.0
sim5                    lx-amd64        7    1    7    7  0.79   23.5G  692.3M    3.0G     0.0
sim6                    lx-amd64        7    1    7    7  0.79   23.5G  692.3M    3.0G     0.0
sim7                    lx-amd64        7    1    7    7  0.79   23.5G  692.3M    3.0G     0.0
sim8                    lx-amd64        7    1    7    7  0.79   23.5G  692.3M    3.0G     0.0
sim9                    lx-amd64        7    1    7    7  0.79   23.5G  692.3M    3.0G     0.0
root@master:~/go/src/github.com/hpc-gridware/go-clusterscheduler/cmd/simulator# qsub -t 1-100:1 -b y sleep 10
Your job-array 1.1-100:1 ("sleep") has been submitted
root@master:~/go/src/github.com/hpc-gridware/go-clusterscheduler/cmd/simulator# qstat
job-ID  prior   name       user         state submit/start at     queue                          slots ja-task-ID
-----------------------------------------------------------------------------------------------------------------
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim3                         1 1
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim4                         1 2
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim5                         1 3
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim2                         1 4
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@master                       1 5
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim9                         1 6
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim7                         1 7
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim1                         1 8
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim6                         1 9
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim8                         1 10
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim8                         1 11
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim6                         1 12
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim1                         1 13
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim7                         1 14
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim9                         1 15
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@master                       1 16
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim2                         1 17
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim5                         1 18
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim4                         1 19
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim3                         1 20
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim3                         1 21
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim4                         1 22
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim5                         1 23
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim2                         1 24
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@master                       1 25
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim9                         1 26
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim7                         1 27
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim1                         1 28
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim6                         1 29
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim8                         1 30
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim8                         1 31
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim6                         1 32
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim1                         1 33
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim7                         1 34
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim9                         1 35
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@master                       1 36
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim2                         1 37
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim5                         1 38
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim4                         1 39
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim3                         1 40
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim3                         1 41
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim4                         1 42
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim5                         1 43
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@sim2                         1 44
      1 0.55500 sleep      root         r     2024-09-02 07:59:27 gpu.q@master                       1 45
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim9                         1 46
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim7                         1 47
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim1                         1 48
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim6                         1 49
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim8                         1 50
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim8                         1 51
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim6                         1 52
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim1                         1 53
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim7                         1 54
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim9                         1 55
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@master                       1 56
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim2                         1 57
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim5                         1 58
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim4                         1 59
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim3                         1 60
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim3                         1 61
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim4                         1 62
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim5                         1 63
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim2                         1 64
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@master                       1 65
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim9                         1 66
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim7                         1 67
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim1                         1 68
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim6                         1 69
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim8                         1 70
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim8                         1 71
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim6                         1 72
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim1                         1 73
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim7                         1 74
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim9                         1 75
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@master                       1 76
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim2                         1 77
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim5                         1 78
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim4                         1 79
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim3                         1 80
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim3                         1 81
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim4                         1 82
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim5                         1 83
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim2                         1 84
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@master                       1 85
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim9                         1 86
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim7                         1 87
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim1                         1 88
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim6                         1 89
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim8                         1 90
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim8                         1 91
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim6                         1 92
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim1                         1 93
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim7                         1 94
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim9                         1 95
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@master                       1 96
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim2                         1 97
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim5                         1 98
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim4                         1 99
      1 0.55500 sleep      root         r     2024-09-02 07:59:28 gpu.q@sim3                         1 100
```
