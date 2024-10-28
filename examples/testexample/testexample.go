package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	qacct "github.com/hpc-gridware/go-clusterscheduler/pkg/qacct/v9.0"
	qstat "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.uber.org/zap"
)

var qacctClient qacct.QAcct
var qstatClient qstat.QStat

var log *zap.Logger

func init() {
	var err error
	log, _ = zap.NewProduction()
	qacctClient, err = qacct.NewCommandLineQAcct(qacct.CommandLineQAcctConfig{})
	if err != nil {
		log.Fatal("Failed to initialize qacct client", zap.String("error",
			err.Error()))
	}
	qstatClient, err = qstat.NewCommandLineQstat(qstat.CommandLineQStatConfig{})
	if err != nil {
		log.Fatal("Failed to initialize qstat client", zap.String("error",
			err.Error()))
	}
}

func main() {
	run(context.Background())
}

func run(ctx context.Context) {
	defer log.Sync()
	alreadySent := map[string]struct{}{}

	for {
		select {
		case <-ctx.Done():
			log.Info("Context cancelled, stopping ClusterScheduler")
			return
		default:
			finishedJobs, err := GetFinishedJobs()
			if err != nil {
				log.Error("Error getting finished jobs", zap.String("error",
					err.Error()))
			}

			runningJobs, err := GetRunningJobs()
			if err != nil {
				log.Error("Error getting running jobs", zap.String("error",
					err.Error()))
			}

			allJobs := append(finishedJobs, runningJobs...)

			var protoJobs []*SimpleJob
			for _, job := range allJobs {
				if _, ok := alreadySent[job.JobId]; ok {
					continue
				}
				protoJobs = append(protoJobs, job)
			}

			_, err = SendJobs(ctx, protoJobs)
			if err != nil {
				log.Warn("Error ingesting jobs", zap.String("error",
					err.Error()))
			}

			for _, job := range finishedJobs {
				alreadySent[job.JobId] = struct{}{}
			}

			select {
			case <-ctx.Done():
				log.Info("Context cancelled, stopping")
				return
			case <-time.After(10 * time.Second):
			}
		}
	}
}

type SimpleJob struct {
	JobId string `json:"job_id"`
	// Cluster represents the queue name
	Cluster string `json:"cluster"`
	JobName string `json:"job_name"`
	// Partition represents the parallel environment
	Partition  string                 `json:"partition"`
	Account    string                 `json:"account"`
	User       string                 `json:"user"`
	State      string                 `json:"state"`
	ExitCode   string                 `json:"exit_code"`
	Submit     *timestamppb.Timestamp `json:"submit"`
	Start      *timestamppb.Timestamp `json:"start"`
	End        *timestamppb.Timestamp `json:"end"`
	MasterNode string                 `json:"master_node"`
}

func GetFinishedJobs() ([]*SimpleJob, error) {
	// Use qacct NativeSpecification to get finished jobs
	qacctOutput, err := qacctClient.NativeSpecification([]string{"-j", "*"})
	if err != nil {
		return nil, fmt.Errorf("error running qacct command: %v", err)
	}

	jobs, err := qacct.ParseQAcctJobOutput(qacctOutput)
	if err != nil {
		return nil, fmt.Errorf("error parsing qacct output: %v", err)
	}
	// convert to SimpleJob format
	simpleJobs := make([]*SimpleJob, len(jobs))
	for i, job := range jobs {
		state := fmt.Sprintf("%d", job.ExitStatus)
		if state == "0" {
			state = "done"
		} else {
			state = "failed"
		}
		simpleJobs[i] = &SimpleJob{
			// ignore job arrays for now
			JobId:      fmt.Sprintf("%d", job.JobNumber),
			Cluster:    job.QName,
			JobName:    job.JobName,
			Partition:  job.GrantedPE,
			Account:    job.Account,
			User:       job.Owner,
			State:      state,
			ExitCode:   fmt.Sprintf("%d", job.ExitStatus),
			Submit:     parseTimestamp(job.QSubTime),
			Start:      parseTimestamp(job.StartTime),
			End:        parseTimestamp(job.EndTime),
			MasterNode: job.HostName,
		}
	}
	return simpleJobs, nil
}

func GetRunningJobs() ([]*SimpleJob, error) {

	qstatOverview, err := qstatClient.NativeSpecification([]string{"-g", "t"})
	if err != nil {
		return nil, fmt.Errorf("error running qstat command: %v", err)
	}
	jobsByTask, err := qstat.ParseGroupByTask(qstatOverview)
	if err != nil {
		return nil, fmt.Errorf("error parsing qstat output: %v", err)
	}

	type State struct {
		QueueName  string
		State      string
		MasterNode string
	}

	stateMap := map[int]State{}

	for _, job := range jobsByTask {
		// we are only interested in serial and parallel jobs (no arrays)
		jq := strings.Split(job.Queue, "@")
		if len(jq) == 2 {
			js, exists := stateMap[job.JobID]
			if !exists {
				js = State{
					QueueName:  jq[0],
					State:      job.State,
					MasterNode: jq[1],
				}
			}
			if job.Master == "MASTER" {
				// this is the master task of a parallel job
				js.MasterNode = jq[1]
				stateMap[job.JobID] = js
			}
			continue
		}
		stateMap[job.JobID] = State{
			QueueName: job.Queue,
			State:     job.State,
		}
	}

	// get running jobs
	qstatOutput, err := qstatClient.NativeSpecification([]string{"-j", "*"})
	if err != nil {
		return nil, fmt.Errorf("error running qstat command: %v", err)
	}

	jobs, err := qstat.ParseSchedulerJobInfo(qstatOutput)
	if err != nil {
		return nil, fmt.Errorf("error parsing qstat output: %v", err)
	}

	// convert to SimpleJob format
	simpleJobs := make([]*SimpleJob, len(jobs))
	for i, job := range jobs {
		state := stateMap[job.JobNumber].State
		if state == "" {
			state = "running"
		}
		simpleJobs[i] = &SimpleJob{
			JobId:      fmt.Sprintf("%d", job.JobNumber),
			Cluster:    stateMap[job.JobNumber].QueueName,
			JobName:    job.JobName,
			Partition:  strings.Split(job.ParallelEnvironment, " ")[0], // PE
			Account:    job.Account,
			User:       job.Owner,
			State:      state,
			ExitCode:   "",
			MasterNode: stateMap[job.JobNumber].MasterNode,
		}
		if strings.Contains(stateMap[job.JobNumber].State, "q") {
			simpleJobs[i].Submit = parseTimestamp(job.SubmissionTime)
		} else {
			simpleJobs[i].Start = parseTimestamp(job.SubmissionTime)
		}
	}
	return simpleJobs, nil
}

func SendJobs(ctx context.Context, jobs []*SimpleJob) (int, error) {
	log.Info("Sending jobs", zap.Int("jobs", len(jobs)))
	// Print the jobs
	for _, job := range jobs {
		// pretty print JSON
		json, err := json.MarshalIndent(job, "", "  ")
		if err != nil {
			return 0, fmt.Errorf("error marshalling job: %v", err)
		}
		fmt.Println(string(json))
	}
	return len(jobs), nil
}

// 2024-10-24 09:49:59.911136
func parseTimestamp(s string) *timestamppb.Timestamp {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return nil
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05.999999", s, loc)
	if err != nil {
		return nil
	}
	return timestamppb.New(t)
}
