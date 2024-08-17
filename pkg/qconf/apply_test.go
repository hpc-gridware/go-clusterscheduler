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

package qconf_test

import (
	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Apply", func() {

	Context("AddAllElements", func() {

		It("should add all elements in correct order", func() {
			// Setup initial and new ClusterConfig instances
			add := qconf.ClusterConfig{
				ComplexEntries: []qconf.ComplexEntryConfig{
					{
						Name:        "addedComplex",
						Shortcut:    "addedc",
						Type:        "INT",
						Relop:       "<=",
						Requestable: "YES",
						Consumable:  "YES",
						Default:     "0",
						Urgency:     1000,
					},
				},
				Calendars: []qconf.CalendarConfig{
					{
						Name: "InitialCalendar1",
						Year: "1.1.2024,6.1.2024,28.3.2024,30.3.2024-31.3.2024,18.5.2024-19.5.2024,3.10.2024,25.12.2024,26.12.2024=on",
					},
				},
				ClusterQueues: []qconf.ClusterQueueConfig{
					{
						Name:      "InitialQueue1",
						Slots:     1,
						XProjects: "NewProject1",
					},
				},
				Projects: []qconf.ProjectConfig{
					{
						Name: "NewProject1",
						ACL:  "UserList1",
					},
				},
				UserSetLists: []qconf.UserSetListConfig{
					{
						Name:    "UserList1",
						Type:    "ACL",
						Entries: "root peter",
					},
				},

				// Initialize other fields as necessary
			}

			qc, err := qconf.NewCommandLineQConf(
				qconf.CommandLineQConfConfig{
					Executable: "qconf",
				})
			Expect(err).ToNot(HaveOccurred())

			added, err := qconf.AddAllEntries(qc, add)
			Expect(err).ToNot(HaveOccurred())

			// GlobalConfig and ClusterEnvironment config cannot be added
			added.GlobalConfig = add.GlobalConfig
			added.ClusterEnvironment = add.ClusterEnvironment

			// Expect that all elements are added
			Expect(added).To(Equal(add))

			// Cleanup
			deleted, err := qconf.DeleteAllEnries(qc, added)
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(Equal(add))
		})

	})

})
