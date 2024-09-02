/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024 HPC-Gridware GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
************************************************************************/
/*___INFO__MARK_END__*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf"
	"github.com/spf13/cobra"
)

func dump(cmd *cobra.Command, args []string) {
	cs, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{
		Executable: "qconf",
		// be friendly and prevent too many requests on the qmaster
		DelayAfter: time.Millisecond * 50,
	})
	FatalOnError(err)

	clusterConfig, err := cs.GetClusterConfiguration()
	FatalOnError(err)

	prettyPrint(clusterConfig)
}

func prettyPrint(v interface{}) {
	js, err := json.MarshalIndent(v, "", "  ")
	FatalOnError(err)
	fmt.Println(string(js))
}

func FatalOnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func PrintOnError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
