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

var _ = Describe("CompareTo", func() {

	Context("Basic functional tests", func() {

		It("should correctly identify added, modified, and removed entries", func() {
			// Setup initial and new ClusterConfig instances
			initial := qconf.ClusterConfig{
				ComplexEntries: []qconf.ComplexEntryConfig{
					{Name: "InitialComplex1"},
					{Name: "InitialComplex2"},
				},
				Calendars: []qconf.CalendarConfig{
					{Name: "InitialCalendar1"},
				},
				AdminHosts: []string{"InitialAdminHost1", "InitialAdminHost2"},
				// Initialize other fields as necessary
			}

			new := qconf.ClusterConfig{
				ComplexEntries: []qconf.ComplexEntryConfig{
					{Name: "InitialComplex1"},
					{Name: "NewComplex3"},
				},
				Calendars: []qconf.CalendarConfig{
					{
						Name: "InitialCalendar1",                                                                                       // update
						Year: "1.1.1999,6.1.1999,28.3.1999,30.3.1999-31.3.1999,18.5.1999-19.5.1999,3.10.1999,25.12.1999,26.12.1999=on", // party time
					},
					{Name: "NewCalendar2"},
				},
				AdminHosts: []string{"NewAdminHost1", "InitialAdminHost1"}, // update
				// Initialize other fields as necessary
			}

			// Call CompareTo function
			comparison, err := initial.CompareTo(new)
			Expect(err).NotTo(HaveOccurred())

			// Validate the DiffAdded section
			Expect(comparison.DiffAdded.ComplexEntries).To(HaveLen(1))
			Expect(comparison.DiffAdded.ComplexEntries[0].Name).To(Equal("NewComplex3"))

			Expect(comparison.DiffAdded.Calendars).To(HaveLen(1))
			Expect(comparison.DiffAdded.Calendars[0].Name).To(Equal("NewCalendar2"))

			Expect(comparison.DiffAdded.AdminHosts).To(HaveLen(1))
			Expect(comparison.DiffAdded.AdminHosts[0]).To(Equal("NewAdminHost1"))

			// Validate the DiffModified section
			Expect(comparison.DiffModified.Calendars).To(HaveLen(1))
			Expect(comparison.DiffModified.Calendars[0].Name).To(Equal("InitialCalendar1"))
			Expect(comparison.DiffModified.Calendars[0].Year).To(Equal("1.1.1999,6.1.1999,28.3.1999,30.3.1999-31.3.1999,18.5.1999-19.5.1999,3.10.1999,25.12.1999,26.12.1999=on"))

			// Validate the DiffRemoved section
			Expect(comparison.DiffRemoved.ComplexEntries).To(HaveLen(1))
			Expect(comparison.DiffRemoved.ComplexEntries[0].Name).To(Equal("InitialComplex2"))
			Expect(comparison.DiffRemoved.AdminHosts).To(HaveLen(1))
			Expect(comparison.DiffRemoved.AdminHosts[0]).To(Equal("InitialAdminHost2"))

			// Additional assertions for other fields if necessary
		})

		It("should correctly identify added, modified, deleted entries", func() {
			qc, err := qconf.NewCommandLineQConf(
				qconf.CommandLineQConfConfig{
					Executable: "qconf",
				})
			Expect(err).To(BeNil())

			cc, err := qc.GetClusterConfiguration()
			Expect(err).To(BeNil())

			emptyConfig := qconf.ClusterConfig{}

			comparison, err := emptyConfig.CompareTo(cc)
			Expect(err).NotTo(HaveOccurred())

			// a lot is added in a default installation ...
			Expect(comparison.DiffAdded).NotTo(BeNil())
			Expect(len(comparison.DiffAdded.ComplexEntries)).To(BeNumerically(">", 10))
			Expect(len(comparison.DiffAdded.ExecHosts)).To(BeNumerically("==", len(cc.ExecHosts)))
			Expect(len(comparison.DiffAdded.HostGroups)).To(BeNumerically("==", len(cc.HostGroups)))

			// 0 if no job has been submitted in this container installation
			// 1 if root submitted a job already
			Expect(len(comparison.DiffAdded.Users)).To(BeNumerically(">=", 0))
			Expect(len(comparison.DiffAdded.Managers)).To(BeNumerically("==", 1))
			Expect(len(comparison.DiffAdded.Operators)).To(BeNumerically("==", 0))

			Expect(comparison.DiffModified).NotTo(BeNil())
			Expect(comparison.DiffModified.GlobalConfig).To(Equal(cc.GlobalConfig))

			// Now the other way around

			comparison, err = cc.CompareTo(emptyConfig)
			Expect(err).NotTo(HaveOccurred())

			// nothing is added to the default installation
			Expect(comparison.DiffAdded).NotTo(BeNil())
			Expect(len(comparison.DiffAdded.ComplexEntries)).To(BeNumerically("==", 0))
			Expect(len(comparison.DiffAdded.ExecHosts)).To(BeNumerically("==", 0))
			Expect(len(comparison.DiffAdded.HostGroups)).To(BeNumerically("==", 0))
			Expect(len(comparison.DiffAdded.Users)).To(BeNumerically("==", 0))
			Expect(len(comparison.DiffAdded.Managers)).To(BeNumerically("==", 0))
			Expect(len(comparison.DiffAdded.Operators)).To(BeNumerically("==", 0))

			// a lot would be removed from the default installation ...
			Expect(comparison.DiffRemoved).NotTo(BeNil())
			Expect(len(comparison.DiffRemoved.ComplexEntries)).To(BeNumerically(">=", 10))
			Expect(len(comparison.DiffRemoved.ExecHosts)).To(BeNumerically(">=", 1))
			Expect(len(comparison.DiffRemoved.HostGroups)).To(BeNumerically(">=", 1))
			Expect(len(comparison.DiffRemoved.Users)).To(BeNumerically(">=", 0))
			Expect(len(comparison.DiffRemoved.Managers)).To(BeNumerically(">=", 1))

			// nothing is added to the default installation
			Expect(comparison.DiffAdded).NotTo(BeNil())
			Expect(len(comparison.DiffAdded.ComplexEntries)).To(BeNumerically("==", 0))
			Expect(len(comparison.DiffAdded.ExecHosts)).To(BeNumerically("==", 0))
			Expect(len(comparison.DiffAdded.HostGroups)).To(BeNumerically("==", 0))
			Expect(len(comparison.DiffAdded.Users)).To(BeNumerically("==", 0))
			Expect(len(comparison.DiffAdded.Managers)).To(BeNumerically("==", 0))

			// Only global configuration is changed
			Expect(comparison.DiffModified).NotTo(BeNil())
			Expect(comparison.DiffModified.GlobalConfig).To(Equal(emptyConfig.GlobalConfig))
		})

	})

})
