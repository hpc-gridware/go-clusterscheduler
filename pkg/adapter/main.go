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
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf"
)

func main() {
	qc, err := qconf.NewCommandLineQConf(
		qconf.CommandLineQConfConfig{Executable: "qconf", DryRun: false})
	if err != nil {
		log.Fatalf("Error creating qconf: %v", err)
	}

	router := mux.NewRouter()
	router.Handle("/api/v0/command", NewAdapter(qc)).Methods("POST")

	log.Println("Starting server on port 8282")
	http.ListenAndServe(":8282", router)
}
