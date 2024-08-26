package main

import (
	"encoding/json"
	"fmt"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf"
)

func main() {
	qc, _ := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{
		Executable: "qconf",
	})
	cc, err := qc.GetClusterConfiguration()
	if err != nil {
		fmt.Println(err)
		return
	}

	// copy - to be replaced by the actual update
	update := qconf.ClusterConfig{}
	bcc, err := json.Marshal(cc)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bcc, &update)
	if err != nil {
		panic(err)
	}

	// add a complex entry
	update.ComplexEntries = append(update.ComplexEntries, qconf.ComplexEntryConfig{
		Name:        "addedComplex",
		Shortcut:    "addedc",
		Type:        "INT",
		Relop:       "<=",
		Requestable: "YES",
		Consumable:  "YES",
		Default:     "0",
		Urgency:     1000,
	})

	comparison, err := cc.CompareTo(update)
	if err != nil {
		panic(err)
	}

	prettyPrint(cc)
}

func prettyPrint(v interface{}) {
	js, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(js))
}
