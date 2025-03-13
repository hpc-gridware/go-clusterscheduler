/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024-2025 HPC-Gridware GmbH
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
	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Apply", func() {

	Context("AddAllElements", func() {

		add := qconf.ClusterConfig{
			ComplexEntries: map[string]qconf.ComplexEntryConfig{
				"addedComplex": {
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
			Calendars: map[string]qconf.CalendarConfig{
				"InitialCalendar1": {
					Name: "InitialCalendar1",
					Year: "1.1.2024,6.1.2024,28.3.2024,30.3.2024-31.3.2024,18.5.2024-19.5.2024,3.10.2024,25.12.2024,26.12.2024=on",
				},
			},
			ClusterQueues: map[string]qconf.ClusterQueueConfig{
				"InitialQueue1": {
					Name:      "InitialQueue1",
					Slots:     []string{"1"},
					XProjects: []string{"NewProject1"},
				},
			},
			Projects: map[string]qconf.ProjectConfig{
				"NewProject1": {
					Name: "NewProject1",
					ACL:  []string{"UserList1"},
				},
			},
			UserSetLists: map[string]qconf.UserSetListConfig{
				"UserList1": {
					Name:    "UserList1",
					Type:    "ACL",
					Entries: []string{"root", "peter"},
				},
			},

			// Initialize other fields as necessary
		}

		It("should add all elements in correct order", func() {
			// Setup initial and new ClusterConfig instances

			qc, err := qconf.NewCommandLineQConf(
				qconf.CommandLineQConfConfig{
					Executable: "qconf",
				})
			Expect(err).ToNot(HaveOccurred())

			// first delete them all
			deleted, _ := qconf.DeleteAllEnries(qc, add, true)

			added, err := qconf.AddAllEntries(qc, add)
			Expect(err).ToNot(HaveOccurred())

			// GlobalConfig and ClusterEnvironment config cannot be added
			added.GlobalConfig = add.GlobalConfig
			added.ClusterEnvironment = add.ClusterEnvironment

			// Expect that all elements are added
			Expect(added).To(Equal(add))

			// Cleanup
			deleted, err = qconf.DeleteAllEnries(qc, added, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(deleted).To(Equal(add))
		})

	})

	Context("Modify all entries", func() {

		var qc *qconf.CommandLineQConf
		var err error

		BeforeEach(func() {
			qc, err = qconf.NewCommandLineQConf(
				qconf.CommandLineQConfConfig{
					Executable: "true",
				})
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {

		})

		It("should modify the global configuration", func() {
			modified, err := qconf.ModifyAllEntries(qc, qconf.ClusterConfig{
				GlobalConfig: &qconf.GlobalConfig{
					MaxJobs: 1000,
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(modified.GlobalConfig).ToNot(BeNil())
			Expect(modified.GlobalConfig.MaxJobs).To(Equal(1000))
			// scheduler configuration should not be modified
			Expect(modified.SchedulerConfig).To(BeNil())
		})

		It("should modify the scheduler configuration", func() {
			modified, err := qconf.ModifyAllEntries(qc, qconf.ClusterConfig{
				SchedulerConfig: &qconf.SchedulerConfig{
					Params: []string{"PROFILE=1", "MONITOR=1"},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(modified.SchedulerConfig).ToNot(BeNil())
			Expect(modified.SchedulerConfig.Params).NotTo(BeNil())
			// global configuration should not be modified
			Expect(modified.GlobalConfig).To(BeNil())
		})

	})

})
