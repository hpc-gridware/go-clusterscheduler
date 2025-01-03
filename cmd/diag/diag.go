package main

import (
	"encoding/json"
	"fmt"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
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
	prettyPrint(cc)
}

func prettyPrint(v interface{}) {
	js, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(js))
}
