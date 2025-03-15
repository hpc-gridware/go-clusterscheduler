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
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qconf "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/v9.0"
)

var _ = Describe("QconfImpl", func() {

	Context("Helper functions", func() {

		It("should run a cli comand", func() {
			ls, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "ls"})
			Expect(err).To(BeNil())
			Expect(ls).NotTo(BeNil())
			out, err := ls.RunCommand("-lisa")
			Expect(err).To(BeNil())
			Expect(out).NotTo(BeNil())
			Expect(out).To(ContainSubstring("total"))
		})

		It("should parse multi-line values", func() {
			lines := strings.Split(`execd_params                 none
reporting_params             accounting=true reporting=false flush_time= 00:00:15 \
                             joblog=true sharelog=00:00:00
finished_jobs                0
gid_range                    20000-20100`, "\n")

			value, isMultiline := qconf.ParseMultiLineValue(lines, 1)
			Expect(isMultiline).To(BeTrue())
			Expect(value).To(Equal("accounting=true reporting=false flush_time= 00:00:15 joblog=true sharelog=00:00:00"))

			value, isMultiline = qconf.ParseMultiLineValue(lines, 0)
			Expect(isMultiline).To(BeFalse())
			Expect(value).To(Equal("none"))
		})

	})

	Context("Cluster configuration", func() {

		It("should read the current cluster configuration", func() {
			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			cc, err := qc.GetClusterConfiguration()
			Expect(err).To(BeNil())
			Expect(cc).NotTo(BeNil())

			/*
				SGE_ROOT=/opt/cs-install
				SGE_CELL=default
				SGE_CLUSTER_NAME=p6444
				SGE_QMASTER_PORT=6444
				SGE_EXECD_PORT=6445
			*/

			env := cc.ClusterEnvironment
			Expect(env.Root).To(Equal("/opt/cs-install"))
			Expect(env.Cell).To(Equal("default"))
			Expect(env.Name).To(Equal("p6444"))
			Expect(env.QmasterPort).To(Equal(6444))
			Expect(env.ExecdPort).To(Equal(6445))
		})

	})

	Context("Resource configuration", func() {

		It("should show, add, list, modify, and delete resources", func() {
			complexName := "go-qconf-impl-test"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			ces, err := qc.ShowComplexEntries()
			Expect(err).To(BeNil())
			Expect(ces).NotTo(BeNil())
			// ces is []string
			Expect(ces).To(ContainElement("arch"))
			Expect(ces).To(ContainElement("calendar"))
			Expect(ces).To(ContainElement("cpu"))
			Expect(ces).To(ContainElement("load_avg"))
			Expect(ces).To(ContainElement("hostname"))

			ce := qconf.ComplexEntryConfig{
				Name:        complexName,
				Shortcut:    "t",
				Type:        "STRING",
				Relop:       "==",
				Requestable: "YES",
				Consumable:  "NO",
				Default:     "NONE",
				Urgency:     0,
			}

			err = qc.AddComplexEntry(ce)
			Expect(err).To(BeNil())

			ces, err = qc.ShowComplexEntries()
			Expect(err).To(BeNil())
			Expect(ces).NotTo(BeNil())
			Expect(ces).To(ContainElement(complexName))

			tc, err := qc.ShowComplexEntry(complexName)
			Expect(err).To(BeNil())
			Expect(tc).To(Equal(ce))

			ce.Shortcut = "tt"
			err = qc.ModifyComplexEntry(complexName, ce)
			Expect(err).To(BeNil())

			tc, err = qc.ShowComplexEntry(complexName)
			Expect(err).To(BeNil())
			Expect(tc).To(Equal(ce))

			err = qc.DeleteComplexEntry(complexName)
			Expect(err).To(BeNil())

			ces, err = qc.ShowComplexEntries()
			Expect(err).To(BeNil())
			Expect(ces).NotTo(BeNil())
			Expect(ces).NotTo(ContainElement(complexName))

			_, err = qc.ShowComplexEntry(complexName)
			Expect(err).NotTo(BeNil())

			ce.Shortcut = "tt"

			all, err := qc.ShowAllComplexes()
			Expect(err).To(BeNil())

			err = qc.ModifyAllComplexes(all)
			Expect(err).To(BeNil())
		})
	})

	Context("Calendar configuration", func() {

		It("should show, add, list, and delete calendars", func() {
			calendarName := "go-qconf-calendar-test"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all calendars, initially should not contain the new calendar
			calendars, err := qc.ShowCalendars()
			Expect(err).To(BeNil())
			Expect(calendars).NotTo(BeNil())
			Expect(calendars).NotTo(ContainElement(calendarName))

			// Define a new calendar configuration
			calConfig := qconf.CalendarConfig{
				Name: calendarName,
				Year: "12.03.2004=12-11=off",
				Week: "NONE",
			}

			// Add the new calendar
			err = qc.AddCalendar(calConfig)
			Expect(err).To(BeNil())

			// Show all calendars, now should contain the new calendar
			calendars, err = qc.ShowCalendars()
			Expect(err).To(BeNil())
			Expect(calendars).NotTo(BeNil())
			Expect(calendars).To(ContainElement(calendarName))

			// Show the specific calendar and verify its configuration
			retrievedCalConfig, err := qc.ShowCalendar(calendarName)
			Expect(err).To(BeNil())
			Expect(retrievedCalConfig).To(Equal(calConfig))

			// Modify the calendar configuration
			calConfig.Year = "12.03.2024=12-11=on"
			err = qc.ModifyCalendar(calendarName, calConfig)
			Expect(err).To(BeNil())

			// Show the specific calendar and verify its configuration
			retrievedCalConfig, err = qc.ShowCalendar(calendarName)
			Expect(err).To(BeNil())
			Expect(retrievedCalConfig).To(Equal(calConfig))

			// Delete the calendar
			err = qc.DeleteCalendar(calendarName)
			Expect(err).To(BeNil())

			// Show all calendars, should no longer contain the deleted calendar
			calendars, err = qc.ShowCalendars()
			Expect(err).To(BeNil())
			Expect(calendars).NotTo(BeNil())
			Expect(calendars).NotTo(ContainElement(calendarName))

			// Show the specific calendar, should return an error
			_, err = qc.ShowCalendar(calendarName)
			Expect(err).NotTo(BeNil())
		})
	})

	It("should show, add, list, and delete checkpointing interfaces", func() {
		interfaceName := "go-qconf-ckpt-test"

		qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
		Expect(err).To(BeNil())

		// Show all checkpointing interfaces, initially should not contain the new interface
		interfaces, err := qc.ShowCkptInterfaces()
		Expect(err).To(BeNil())
		Expect(interfaces).NotTo(BeNil())
		Expect(interfaces).NotTo(ContainElement(interfaceName))

		// Define a new checkpointing interface configuration
		ckptConfig := qconf.CkptInterfaceConfig{
			Name:              interfaceName,
			Interface:         "userdefined",
			CleanCommand:      "/path/to/clean_command",
			CheckpointCommand: "/path/to/checkpoint_cmd",
			MigrCommand:       "/path/to/migr_command",
			RestartCommand:    "/path/to/restart_command",
			CheckpointDir:     "/path/to/ckpt_dir",
			When:              "xmr",
			Signal:            "usr2",
		}

		// Add the new checkpointing interface
		err = qc.AddCkptInterface(ckptConfig)
		Expect(err).To(BeNil())

		// Show all checkpointing interfaces, now should contain the new interface
		interfaces, err = qc.ShowCkptInterfaces()
		Expect(err).To(BeNil())
		Expect(interfaces).NotTo(BeNil())
		Expect(interfaces).To(ContainElement(interfaceName))

		// Show the specific checkpointing interface and verify its configuration
		retrievedCkptConfig, err := qc.ShowCkptInterface(interfaceName)
		Expect(err).To(BeNil())
		Expect(retrievedCkptConfig).To(Equal(ckptConfig))

		// Modify the checkpointing interface configuration
		ckptConfig.CleanCommand = "/path/to/modified_clean_command"
		err = qc.ModifyCkptInterface(interfaceName, ckptConfig)
		Expect(err).To(BeNil())

		// Show the specific checkpointing interface and verify its configuration
		retrievedCkptConfig, err = qc.ShowCkptInterface(interfaceName)
		Expect(err).To(BeNil())
		Expect(retrievedCkptConfig).To(Equal(ckptConfig))

		// Delete the checkpointing interface
		err = qc.DeleteCkptInterface(interfaceName)
		Expect(err).To(BeNil())

		// Show all checkpointing interfaces, should no longer contain the deleted interface
		interfaces, err = qc.ShowCkptInterfaces()
		Expect(err).To(BeNil())
		Expect(interfaces).NotTo(BeNil())
		Expect(interfaces).NotTo(ContainElement(interfaceName))

		// Show the specific checkpointing interface, should return an error
		_, err = qc.ShowCkptInterface(interfaceName)
		Expect(err).NotTo(BeNil())
	})

	Context("Global configuration", func() {

		var qc *qconf.CommandLineQConf
		var err error

		var global *qconf.GlobalConfig

		BeforeEach(func() {
			qc, err = qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{
				Executable: "qconf"})
			Expect(err).To(BeNil())
			global, err = qc.ShowGlobalConfiguration()
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			qc.ModifyGlobalConfig(*global)
		})

		It("should show and modify the global configuration", func() {

			// Retrieve the current global configuration
			globalConfig, err := qc.ShowGlobalConfiguration()
			Expect(err).To(BeNil())
			Expect(globalConfig).NotTo(BeNil())

			// Modify the global configuration for the test
			modifiedConfig := globalConfig
			modifiedConfig.ExecdSpoolDir = "/new/spool/dir"
			modifiedConfig.AutoUserDeleteTime = 3600
			modifiedConfig.AdministratorMail = "admin@example.com"
			modifiedConfig.QmasterParams = []string{"ENABLE_FORCED_QDEL_IF_UNKNOWN=true", "MONITOR_TIME=0:0:10"}
			modifiedConfig.ExecdParams = []string{"KEEP_ACTIVE=true", "USE_QSUB_GID=true"}
			modifiedConfig.GidRange = []string{"20000-20100", "20101-20200"}
			modifiedConfig.JsvAllowedMod = []string{"l_hard", "l_soft"}
			modifiedConfig.UserLists = []string{"arusers", "deadlineusers"}
			modifiedConfig.ReportingParams = []string{"accounting_flush_time=00:00:00", "joblog=true"}
			modifiedConfig.LoginShells = []string{"/bin/bash", "/bin/zsh", "/bin/sh"}
			//modifiedConfig.XProjects = []string{"p1", "p2"}

			// Apply the modified configuration
			err = qc.ModifyGlobalConfig(*modifiedConfig)
			Expect(err).To(BeNil())

			// Verify that the global configuration was correctly modified
			retrievedConfig, err := qc.ShowGlobalConfiguration()
			Expect(err).To(BeNil())
			Expect(retrievedConfig).To(Equal(modifiedConfig))
		})

		It("should handle bool and int fields correctly when modifying global configuration", func() {
			// Retrieve the current global configuration
			globalConfig, err := qc.ShowGlobalConfiguration()
			Expect(err).To(BeNil())
			Expect(globalConfig).NotTo(BeNil())

			// Modify the global configuration with emphasis on bool and int fields
			modifiedConfig := globalConfig
			modifiedConfig.EnforceProject = "true"
			modifiedConfig.MaxJobs = globalConfig.MaxJobs + 1

			// Apply the modified configuration
			err = qc.ModifyGlobalConfig(*modifiedConfig)
			Expect(err).To(BeNil())

			// Verify that the bool and int fields are modified correctly
			retrievedConfig, err := qc.ShowGlobalConfiguration()
			Expect(err).To(BeNil())
			Expect(retrievedConfig.EnforceProject).To(Equal(modifiedConfig.EnforceProject))
			Expect(retrievedConfig.MaxJobs).To(Equal(modifiedConfig.MaxJobs))
			Expect(len(retrievedConfig.LoginShells)).To(BeNumerically("==", 5))
			// none of the slices should be < 1 in length since "NONE" must always be present
		})

	})

	Context("Host configuration", func() {

		It("should show, add, list, and delete host configurations", func() {
			// We expect that this host is part of the cluster
			hostName, _ := os.Hostname()

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all host configurations, initially should not contain the new host
			hosts, err := qc.ShowHostConfigurations()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).To(ContainElement(hostName))

			// Define a new host configuration
			hostConfig := qconf.HostConfiguration{
				Name:   hostName,
				Mailer: "/mailer",
				Xterm:  "/xterm",
				// Add other necessary fields here...
			}

			// Add the new host configuration
			err = qc.ModifyHostConfiguration(hostConfig.Name, hostConfig)
			Expect(err).To(BeNil())

			// Show all host configurations, now should contain the new host
			hosts, err = qc.ShowHostConfigurations()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).To(ContainElement(hostName))

			// Show the specific host configuration and verify its configuration
			retrievedHostConfig, err := qc.ShowHostConfiguration(hostName)
			Expect(err).To(BeNil())
			Expect(retrievedHostConfig).To(Equal(hostConfig))

			// Delete the host configuration
			err = qc.DeleteHostConfiguration(hostName)
			Expect(err).To(BeNil())

			// Show all host configurations, should no longer contain the deleted host
			hosts, err = qc.ShowHostConfigurations()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).NotTo(ContainElement(hostName))

			// Show the specific host configuration, should return an error
			_, err = qc.ShowHostConfiguration(hostName)
			Expect(err).NotTo(BeNil())

			qc.AddHostConfiguration(hostConfig)
		})
	})

	Context("Execution Host configuration", func() {

		var qc *qconf.CommandLineQConf
		var err error

		BeforeEach(func() {
			qc, err = qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			qc.DeleteAttribute("hostgroup", "hostlist", "master", "@allhosts")
			qc.DeleteExecHost("master")
			qc.DeleteComplexEntry("test_mem")

			qc.DeleteProject([]string{"test_project", "test_project2",
				"test_project3", "test_project4"})

			qc.ModifyHostGroup("@allhosts", qconf.HostGroupConfig{
				Name:  "@allhosts",
				Hosts: []string{"master"},
			})
		})

		It("should show, add, list, and delete execution hosts", func() {
			hostName := "exec-host-test"

			// Show all execution hosts, initially should not contain the new host
			hosts, err := qc.ShowExecHosts()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).NotTo(ContainElement(hostName))

			eh, err := qc.ShowExecHost("master")
			Expect(err).To(BeNil())
			Expect(eh).NotTo(BeNil())

			allhosts, err := qc.ShowHostGroup("@allhosts")
			Expect(err).To(BeNil())

			Expect(allhosts).NotTo(BeNil())
			Expect(allhosts.Hosts).To(ContainElement("master"))

			err = qc.AddComplexEntry(qconf.ComplexEntryConfig{
				Name:        "test_mem",
				Shortcut:    "tm",
				Type:        "INT",
				Relop:       "<=",
				Requestable: "YES",
				Default:     "0",
				Urgency:     0,
			})
			Expect(err).To(BeNil())

			// set on host
			eh.ComplexValues = map[string]string{"test_mem": "1024"}
			eh.UserLists = []string{"arusers", "deadlineusers"}
			eh.XUserLists = nil
			err = qc.ModifyExecHost("master", eh)
			Expect(err).To(BeNil())

			eh, err = qc.ShowExecHost("master")
			Expect(err).To(BeNil())
			Expect(eh).NotTo(BeNil())

			Expect(eh.ComplexValues).To(
				Equal(map[string]string{"test_mem": "1024"}))
			Expect(eh.UserLists).To(Equal([]string{"arusers", "deadlineusers"}))
			Expect(eh.XUserLists).To(BeNil())

			err = qc.DeleteAttribute("exechost", "complex_values", "test_mem", "master")
			Expect(err).To(BeNil())

			eh, err = qc.ShowExecHost("master")
			Expect(err).To(BeNil())
			Expect(eh).NotTo(BeNil())

			Expect(len(eh.ComplexValues)).To(Equal(0))

			err = qc.DeleteComplexEntry("test_mem")
			Expect(err).To(BeNil())

			// Delete exec host config - fails because it is in use
			// by all.q in @allhosts
			err = qc.DeleteExecHost("master")
			// TODO sometimes nil
			Expect(err).NotTo(BeNil())

			// TODO: Method for removing a host from a hostgroup
			// remove from allhosts
			err = qc.DeleteAttribute("hostgroup", "hostlist", "master", "@allhosts")
			Expect(err).To(BeNil())

			// now it should work
			err = qc.DeleteExecHost("master")
			Expect(err).To(BeNil())

			err = qc.AddExecHost(eh)
			Expect(err).To(BeNil())

			err = qc.AddAttribute("hostgroup", "hostlist", "master", "@allhosts")
			Expect(err).To(BeNil())

		})

		It("should show, add, and delete execution hosts", func() {

			qc.AddProject(qconf.ProjectConfig{
				Name: "test_project",
			})
			defer qc.DeleteProject([]string{"test_project"})

			qc.AddProject(qconf.ProjectConfig{
				Name: "test_project2",
			})
			defer qc.DeleteProject([]string{"test_project2"})

			qc.AddProject(qconf.ProjectConfig{
				Name: "test_project3",
			})
			defer qc.DeleteProject([]string{"test_project3"})

			qc.AddProject(qconf.ProjectConfig{
				Name: "test_project4",
			})
			defer qc.DeleteProject([]string{"test_project4"})

			// Show all execution hosts, initially should not contain the new host
			hosts, err := qc.ShowExecHosts()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(len(hosts)).To(BeNumerically(">", 0))

			testhost := hosts[0]

			// Show the specific execution host and verify its configuration
			retrievedHost, err := qc.ShowExecHost(testhost)
			Expect(err).To(BeNil())

			hostConfigCopy := qconf.HostExecConfig{
				Name:            retrievedHost.Name,
				LoadScaling:     retrievedHost.LoadScaling,
				UsageScaling:    retrievedHost.UsageScaling,
				ComplexValues:   retrievedHost.ComplexValues,
				UserLists:       retrievedHost.UserLists,
				XUserLists:      retrievedHost.XUserLists,
				Projects:        retrievedHost.Projects,
				XProjects:       retrievedHost.XProjects,
				ReportVariables: retrievedHost.ReportVariables,
			}

			// Modify the host configuration with multiple entries
			hostConfigCopy.LoadScaling = map[string]float64{
				"np_load_avg":  0.5,
				"np_load_long": 1.2,
			}
			hostConfigCopy.UsageScaling = map[string]float64{
				"cpu": 0.5,
				"mem": 1.2,
				"io":  0.8,
			}
			hostConfigCopy.ReportVariables = []string{"mem_free", "np_load_avg"}
			hostConfigCopy.ComplexValues = map[string]string{
				"mem_free":    "1024",
				"np_load_avg": "4",
			}
			hostConfigCopy.UserLists = []string{"arusers", "defaultdepartment"}
			hostConfigCopy.XUserLists = []string{"deadlineusers"}
			hostConfigCopy.Projects = []string{"test_project", "test_project2"}
			hostConfigCopy.XProjects = []string{"test_project3", "test_project4"}
			err = qc.ModifyExecHost(testhost, hostConfigCopy)
			Expect(err).To(BeNil())

			chc, err := qc.ShowExecHost(testhost)
			Expect(err).To(BeNil())

			Expect(chc.LoadScaling).To(Equal(map[string]float64{
				"np_load_avg":  0.5,
				"np_load_long": 1.2,
			}))
			Expect(chc.UsageScaling).To(Equal(map[string]float64{
				"cpu": 0.5,
				"mem": 1.2,
				"io":  0.8,
			}))
			Expect(chc.ReportVariables).To(Equal([]string{
				"mem_free", "np_load_avg",
			}))
			Expect(chc.ComplexValues).To(Equal(map[string]string{
				"mem_free":    "1024",
				"np_load_avg": "4",
			}))
			Expect(chc.UserLists).To(Equal([]string{"arusers", "defaultdepartment"}))
			Expect(chc.XUserLists).To(Equal([]string{"deadlineusers"}))
			Expect(chc.Projects).To(Equal([]string{"test_project", "test_project2"}))
			Expect(chc.XProjects).To(Equal([]string{"test_project3", "test_project4"}))

			// Modify the host configuration with empty entries
			hostConfigCopy.LoadScaling = nil
			hostConfigCopy.UsageScaling = map[string]float64{}
			hostConfigCopy.ReportVariables = []string{}
			hostConfigCopy.ComplexValues = map[string]string{}
			hostConfigCopy.UserLists = []string{}
			err = qc.ModifyExecHost(testhost, hostConfigCopy)
			Expect(err).To(BeNil())

			// change back to original values
			err = qc.ModifyExecHost(testhost, retrievedHost)
			Expect(err).To(BeNil())
		})
	})

	Context("Admin Host configuration", func() {

		It("should show, add, and delete admin hosts", func() {
			adminHosts := []string{"localhost"}

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all admin hosts, initially should not contain the new hosts
			hosts, err := qc.ShowAdminHosts()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).NotTo(ContainElement(adminHosts[0]))

			// Add the new admin hosts
			err = qc.AddAdminHost(adminHosts)
			Expect(err).To(BeNil())

			// Show all admin hosts, now should contain the new hosts
			hosts, err = qc.ShowAdminHosts()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).To(ContainElement(adminHosts[0]))

			// Delete the admin hosts
			err = qc.DeleteAdminHost(adminHosts)
			Expect(err).To(BeNil())

			// Show all admin hosts, should no longer contain the deleted hosts
			hosts, err = qc.ShowAdminHosts()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).NotTo(ContainElement(adminHosts[0]))
		})

	})

	Context("Host Group configuration", func() {

		var qc *qconf.CommandLineQConf
		var err error

		groupName := "@host-group-test"

		BeforeEach(func() {
			qc, err = qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			qc.DeleteHostGroup(groupName)
		})

		It("should show, add, list, and delete host groups", func() {

			// Show all host groups, initially should not contain the new group
			groups, err := qc.ShowHostGroups()
			Expect(err).To(BeNil())
			Expect(groups).NotTo(BeNil())
			Expect(groups).NotTo(ContainElement(groupName))

			// Define a new host group configuration
			hostGroupConfig := qconf.HostGroupConfig{
				Name:  groupName,
				Hosts: []string{"master"},
			}

			// Add the new host group
			err = qc.AddHostGroup(hostGroupConfig)
			Expect(err).To(BeNil())

			// Show all host groups, now should contain the new group
			groups, err = qc.ShowHostGroups()
			Expect(err).To(BeNil())
			Expect(groups).NotTo(BeNil())
			Expect(groups).To(ContainElement(groupName))

			// Show a list of all hosts in the host group
			hosts, err := qc.ShowHostGroupResolved(groupName)
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).To(ContainElement("master"))

			retrievedHostGroupConfig, err := qc.ShowHostGroup(groupName)
			Expect(err).To(BeNil())
			Expect(retrievedHostGroupConfig).To(Equal(hostGroupConfig))

			retrievedHostGroupConfig.Hosts = nil
			err = qc.ModifyHostGroup(groupName, retrievedHostGroupConfig)
			Expect(err).To(BeNil())

			// Delete the host group configuration
			err = qc.DeleteHostGroup(groupName)
			Expect(err).To(BeNil())

			// Show all host groups, should no longer contain the deleted group
			groups, err = qc.ShowHostGroups()
			Expect(err).To(BeNil())
			Expect(groups).NotTo(BeNil())
			Expect(groups).NotTo(ContainElement(groupName))

			// Show the specific host group configuration, should return an error
			_, err = qc.ShowHostGroup(groupName)
			Expect(err).NotTo(BeNil())
		})
	})

	Context("Resource Quota Set configuration", func() {

		It("should show, add, list, and delete resource quota sets", func() {
			rqsName := "resource-quota-test"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all resource quota sets, initially should not contain the new set
			rqss, err := qc.ShowResourceQuotaSets()
			Expect(err).To(BeNil())
			Expect(rqss).NotTo(BeNil())
			Expect(rqss).NotTo(ContainElement(rqsName))

			// Define a new resource quota set configuration
			rqsConfig := qconf.ResourceQuotaSetConfig{
				Name:        rqsName,
				Description: "Test RQS",
				Enabled:     true,
				Limits:      []string{"users root to slots=5"},
			}

			// Add the new resource quota set
			err = qc.AddResourceQuotaSet(rqsConfig)
			Expect(err).To(BeNil())

			// Show all resource quota sets, now should contain the new set
			rqss, err = qc.ShowResourceQuotaSets()
			Expect(err).To(BeNil())
			Expect(rqss).NotTo(BeNil())
			Expect(rqss).To(ContainElement(rqsName))

			// Show the specific resource quota set configuration and verify its details
			retrievedRqsConfig, err := qc.ShowResourceQuotaSet(rqsName)
			Expect(err).To(BeNil())
			Expect(retrievedRqsConfig).To(Equal(rqsConfig))

			// Modify the resource quota set configuration
			rqsConfig.Description = ""
			rqsConfig.Limits = []string{"users root to slots=11"}
			err = qc.ModifyResourceQuotaSet(rqsName, rqsConfig)
			Expect(err).To(BeNil())

			// Delete the resource quota set configuration
			err = qc.DeleteResourceQuotaSet(rqsName)
			Expect(err).To(BeNil())

			// Show all resource quota sets, should no longer contain the deleted set
			rqss, err = qc.ShowResourceQuotaSets()
			Expect(err).To(BeNil())
			Expect(rqss).NotTo(BeNil())
			Expect(rqss).NotTo(ContainElement(rqsName))

			// Show the specific resource quota set configuration, should return an error
			_, err = qc.ShowResourceQuotaSet(rqsName)
			Expect(err).NotTo(BeNil())
		})

	})

	Context("Manager List configuration", func() {

		It("should show, add, and delete managers", func() {
			managers := []string{"master"}

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all managers, initially should not contain the new manager
			managerList, err := qc.ShowManagers()
			Expect(err).To(BeNil())
			Expect(managerList).NotTo(BeNil())
			Expect(managerList).NotTo(ContainElement(managers[0]))

			// Add the new manager
			err = qc.AddUserToManagerList(managers)
			Expect(err).To(BeNil())

			// Show all managers, now should contain the new manager
			managerList, err = qc.ShowManagers()
			Expect(err).To(BeNil())
			Expect(managerList).NotTo(BeNil())
			Expect(managerList).To(ContainElement(managers[0]))

			// Delete the manager
			err = qc.DeleteUserFromManagerList(managers)
			Expect(err).To(BeNil())

			// Show all managers, should no longer contain the deleted manager
			managerList, err = qc.ShowManagers()
			Expect(err).To(BeNil())
			Expect(managerList).NotTo(BeNil())
			Expect(managerList).NotTo(ContainElement(managers[0]))
		})
	})

	Context("Operator List configuration", func() {

		It("should show, add, and delete operators", func() {
			operators := []string{"master"}

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all operators, initially should not contain the new operator
			operatorList, err := qc.ShowOperators()
			Expect(err).To(BeNil())
			Expect(operatorList).NotTo(BeNil())
			Expect(operatorList).NotTo(ContainElement(operators[0]))

			// Add the new operator
			err = qc.AddUserToOperatorList(operators)
			Expect(err).To(BeNil())

			// Show all operators, now should contain the new operator
			operatorList, err = qc.ShowOperators()
			Expect(err).To(BeNil())
			Expect(operatorList).NotTo(BeNil())
			Expect(operatorList).To(ContainElement(operators[0]))

			// Delete the operator
			err = qc.DeleteUserFromOperatorList(operators)
			Expect(err).To(BeNil())

			// Show all operators, should no longer contain the deleted operator
			operatorList, err = qc.ShowOperators()
			Expect(err).To(BeNil())
			Expect(operatorList).NotTo(BeNil())
			Expect(operatorList).NotTo(ContainElement(operators[0]))
		})
	})

	Context("Parallel Environment configuration", func() {

		It("should show, add, list, and delete parallel environments", func() {
			peName := "parallel-env-test"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all parallel environments, initially should not contain the new PE
			pes, err := qc.ShowParallelEnvironments()
			Expect(err).To(BeNil())
			Expect(pes).NotTo(BeNil())
			Expect(pes).NotTo(ContainElement(peName))

			// Define a new parallel environment configuration
			peConfig := qconf.ParallelEnvironmentConfig{
				Name:              peName,
				Slots:             100,
				UserLists:         []string{"arusers", "defaultdepartment"},
				XUserLists:        []string{"deadlineusers"},
				StartProcArgs:     "/start/procedure",
				StopProcArgs:      "/stop/procedure",
				AllocationRule:    "$pe_slots",
				ControlSlaves:     "TRUE",
				JobIsFirstTask:    true,
				UrgencySlots:      "100",
				AccountingSummary: true,
			}

			// Add the new parallel environment
			err = qc.AddParallelEnvironment(peConfig)
			Expect(err).To(BeNil())

			// Show all parallel environments, now should contain the new PE
			pes, err = qc.ShowParallelEnvironments()
			Expect(err).To(BeNil())
			Expect(pes).NotTo(BeNil())
			Expect(pes).To(ContainElement(peName))

			// Show the specific parallel environment and verify its details
			retrievedPeConfig, err := qc.ShowParallelEnvironment(peName)
			Expect(err).To(BeNil())
			Expect(retrievedPeConfig).To(Equal(peConfig))

			// Modify the parallel environment configuration
			peConfig.Slots = 200
			err = qc.ModifyParallelEnvironment(peName, peConfig)
			Expect(err).To(BeNil())

			// Show the specific parallel environment and verify its details
			// Verify that the configuration has been updated
			retrievedPeConfig, err = qc.ShowParallelEnvironment(peName)
			Expect(err).To(BeNil())
			Expect(retrievedPeConfig).To(Equal(peConfig))

			// Delete the parallel environment
			err = qc.DeleteParallelEnvironment(peName)
			Expect(err).To(BeNil())

			// Show all parallel environments, should no longer contain the deleted PE
			pes, err = qc.ShowParallelEnvironments()
			Expect(err).To(BeNil())
			Expect(pes).NotTo(BeNil())
			Expect(pes).NotTo(ContainElement(peName))

			// Show the specific parallel environment, should return an error
			_, err = qc.ShowParallelEnvironment(peName)
			Expect(err).NotTo(BeNil())
		})
	})

	Context("Project configuration", func() {

		It("should show, add, list, and delete projects", func() {
			projectName := "project-test"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all projects, initially should not contain the new project
			projects, err := qc.ShowProjects()
			Expect(err).To(BeNil())
			Expect(projects).NotTo(BeNil())
			Expect(projects).NotTo(ContainElement(projectName))

			// Define a new project configuration
			projectConfig := qconf.ProjectConfig{
				Name:    projectName,
				OTicket: 111,
				FShare:  222,
				ACL:     nil,
				XACL:    nil,
			}

			// Add the new project
			err = qc.AddProject(projectConfig)
			Expect(err).To(BeNil())

			// Show all projects, now should contain the new project
			projects, err = qc.ShowProjects()
			Expect(err).To(BeNil())
			Expect(projects).NotTo(BeNil())
			Expect(projects).To(ContainElement(projectName))

			// Show the specific project configuration and verify its details
			retrievedProjectConfig, err := qc.ShowProject(projectName)
			Expect(err).To(BeNil())
			qconf.SetDefaultProjectValues(&projectConfig)
			Expect(retrievedProjectConfig).To(Equal(projectConfig))

			// Modify project
			projectConfig.OTicket = 333
			err = qc.ModifyProject(projectName, projectConfig)
			Expect(err).To(BeNil())
			retrievedProjectConfig, err = qc.ShowProject(projectName)
			Expect(err).To(BeNil())
			Expect(retrievedProjectConfig).To(Equal(projectConfig))

			// Delete the project
			err = qc.DeleteProject([]string{projectName})
			Expect(err).To(BeNil())

			// Show all projects, should no longer contain the deleted project
			projects, err = qc.ShowProjects()
			Expect(err).To(BeNil())
			Expect(projects).NotTo(BeNil())
			Expect(projects).NotTo(ContainElement(projectName))

			// Show the specific project configuration, should return an error
			_, err = qc.ShowProject(projectName)
			Expect(err).NotTo(BeNil())
		})
	})

	Context("Cluster Queue configuration", func() {

		var qc *qconf.CommandLineQConf
		var err error

		queueName := "cluster-queue-test"

		BeforeEach(func() {
			qc, err = qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			err = qc.AddParallelEnvironment(
				qconf.ParallelEnvironmentConfig{
					Name: "p1",
				},
			)
			Expect(err).To(BeNil())

			err = qc.AddParallelEnvironment(
				qconf.ParallelEnvironmentConfig{
					Name: "p2",
				},
			)
			Expect(err).To(BeNil())

			err = qc.AddHostGroup(
				qconf.HostGroupConfig{
					Name:  "@newhosts",
					Hosts: []string{},
				})
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			qc.DeleteClusterQueue(queueName)
			qc.DeleteParallelEnvironment("p1")
			qc.DeleteParallelEnvironment("p2")
			qc.DeleteHostGroup("@newhosts")
		})

		It("should show, add, list, and delete cluster queues", func() {

			// Show all cluster queues, initially should not contain the new queue
			queues, err := qc.ShowClusterQueues()
			Expect(err).To(BeNil())
			Expect(queues).NotTo(BeNil())
			Expect(queues).NotTo(ContainElement(queueName))

			queueConfig := qconf.ClusterQueueConfig{
				Name:     queueName,
				HostList: []string{"@allhosts"},
				SeqNo:    []string{"77"},
				Priority: []string{"0"},
				Slots:    []string{"10"},
				PeList:   []string{"p1", "p2"},
				QType:    []string{qconf.QTypeBatch, qconf.QTypeInteractive},
				//ChktList
				OwnerList:     []string{"root"},
				UserLists:     []string{"arusers", "deadlineusers"},
				ComplexValues: []string{"slots=10", "s_rt=86400", "mem_free=10G"},
			}

			// Add the new cluster queue
			err = qc.AddClusterQueue(queueConfig)
			Expect(err).To(BeNil())

			// Show all cluster queues, now should contain the new queue
			queues, err = qc.ShowClusterQueues()
			Expect(err).To(BeNil())
			Expect(queues).NotTo(BeNil())
			Expect(queues).To(ContainElement(queueName))

			// Show the specific cluster queue configuration and verify its details
			retrievedQueueConfig, err := qc.ShowClusterQueue(queueName)
			Expect(err).To(BeNil())
			// add some default values which are not set
			qconf.SetDefaultQueueValues(&queueConfig)
			Expect(retrievedQueueConfig).To(Equal(queueConfig))

			newQueueConfig := qconf.ClusterQueueConfig{
				Name:           queueName,
				HostList:       []string{"@allhosts", "@newhosts"},
				SeqNo:          []string{"99"},
				LoadThresholds: []string{"np_load_avg=1.75"},
				Slots:          []string{"50"},
				MinCpuInterval: []string{"00:01:00"},
				QType:          []string{"BATCH", "INTERACTIVE"},
				Prolog:         []string{"/new/prolog"},
				Epilog:         []string{"/new/epilog"},
				InitialState:   []string{"disabled"},
				Rerun:          []string{"TRUE"},
			}
			err = qc.ModifyClusterQueue(queueName, newQueueConfig)
			Expect(err).To(BeNil())

			// Show the specific cluster queue configuration and verify its details
			retrievedQueueConfig, err = qc.ShowClusterQueue(queueName)
			Expect(err).To(BeNil())
			// add some default values which are not set
			qconf.SetDefaultQueueValues(&newQueueConfig)
			Expect(retrievedQueueConfig).To(Equal(newQueueConfig))

			// Delete the cluster queue
			err = qc.DeleteClusterQueue(queueName)
			Expect(err).To(BeNil())

			// Show all cluster queues, should no longer contain the deleted queue
			queues, err = qc.ShowClusterQueues()
			Expect(err).To(BeNil())
			Expect(queues).NotTo(BeNil())
			Expect(queues).NotTo(ContainElement(queueName))

			// Show the specific cluster queue configuration, should return an error
			_, err = qc.ShowClusterQueue(queueName)
			Expect(err).NotTo(BeNil())
		})
	})

	Context("Submit Host configuration", func() {

		It("should show, add, and delete submit hosts", func() {
			hostname := "localhost"

			submitHosts := []string{hostname}

			qc, err := qconf.NewCommandLineQConf(
				qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all submit hosts, initially should not contain the new hosts
			hosts, err := qc.ShowSubmitHosts()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).NotTo(ContainElement(submitHosts[0]))

			// Add the new submit hosts
			err = qc.AddSubmitHosts(submitHosts)
			Expect(err).To(BeNil())

			// Show all submit hosts, now should contain the new hosts
			hosts, err = qc.ShowSubmitHosts()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).To(ContainElement(submitHosts[0]))

			// Delete the submit hosts
			err = qc.DeleteSubmitHost(submitHosts)
			Expect(err).To(BeNil())

			// Show all submit hosts, should no longer contain the deleted hosts
			hosts, err = qc.ShowSubmitHosts()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).NotTo(ContainElement(submitHosts[0]))
		})

	})

	Context("User Set List configuration", func() {

		listName := "testlist"
		user := "root"
		user2 := "ubuntu"

		var qc qconf.QConf
		var err error

		BeforeEach(func() {
			qc, err = qconf.NewCommandLineQConf(
				qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			qc.DeleteUserSetList(listName)
		})

		It("should show, add users to, list, and delete from user set lists", func() {

			// Show all user set lists, initially should not contain the new list
			lists, err := qc.ShowUserSetLists()
			Expect(err).To(BeNil())
			Expect(lists).NotTo(BeNil())

			// Define a new user set list configuration
			userSetListConfig := qconf.UserSetListConfig{
				Name:    listName,
				Type:    "ACL DEPT",
				FShare:  100,
				OTicket: 10,
			}

			err = qc.AddUserSetList(listName, userSetListConfig)
			Expect(err).To(BeNil())
			// Add user to the user set list
			err = qc.AddUserToUserSetList(user, listName)
			Expect(err).To(BeNil())

			err = qc.AddUserToUserSetList(user2, listName)
			Expect(err).To(BeNil())

			// Show all user set lists, now should contain the new list
			lists, err = qc.ShowUserSetLists()
			Expect(err).To(BeNil())
			Expect(lists).NotTo(BeNil())
			Expect(lists).To(ContainElement(listName))

			// Show the specific user set list configuration and verify its details
			retrievedUserSetListConfig, err := qc.ShowUserSetList(listName)
			Expect(err).To(BeNil())
			userSetListConfig.Entries = []string{user, user2}
			Expect(retrievedUserSetListConfig).To(Equal(userSetListConfig))

			// Delete user from the user set list
			err = qc.DeleteUserFromUserSetList(user, listName)
			Expect(err).To(BeNil())

			retrievedUserSetListConfig, err = qc.ShowUserSetList(listName)
			Expect(err).To(BeNil())
			Expect(len(retrievedUserSetListConfig.Entries)).To(Equal(1))

			// Modify
			userSetListConfig.OTicket = 20
			err = qc.ModifyUserset(listName, userSetListConfig)
			Expect(err).To(BeNil())

			modifiedUserSetListConfig, err := qc.ShowUserSetList(listName)
			Expect(err).To(BeNil())

			Expect(modifiedUserSetListConfig.OTicket).To(Equal(20))

			err = qc.DeleteUserSetList(listName)
			Expect(err).To(BeNil())

			// Show all user set lists, should no longer contain the deleted list
			lists, err = qc.ShowUserSetLists()
			Expect(err).To(BeNil())
			Expect(lists).NotTo(BeNil())
			Expect(lists).NotTo(ContainElement(listName))

			// Show the specific user set list, should return an error
			_, err = qc.ShowUserSetList(listName)
			Expect(err).NotTo(BeNil())
		})
	})

	Context("User configuration", func() {

		It("should show, add, list, and delete users", func() {
			userName := "root"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			qc.DeleteUser([]string{"root"})

			// Show all users, initially should not contain the new user
			users, err := qc.ShowUsers()
			Expect(err).To(BeNil())
			Expect(users).NotTo(BeNil())
			Expect(users).NotTo(ContainElement(userName))

			err = qc.AddProject(qconf.ProjectConfig{Name: "defaultP"})
			Expect(err).To(BeNil())

			// Define a new user configuration
			userConfig := qconf.UserConfig{
				Name:           userName,
				OTicket:        100,
				FShare:         50,
				DeleteTime:     0,
				DefaultProject: "defaultP",
			}

			// Add the new user
			err = qc.AddUser(userConfig)
			Expect(err).To(BeNil())

			// Show all users, now should contain the new user
			users, err = qc.ShowUsers()
			Expect(err).To(BeNil())
			Expect(users).NotTo(BeNil())
			Expect(users).To(ContainElement(userName))

			// Show the specific user configuration and verify its details
			retrievedUserConfig, err := qc.ShowUser(userName)
			Expect(err).To(BeNil())
			Expect(retrievedUserConfig).To(Equal(userConfig))

			newUserConfig := qconf.UserConfig{
				Name:           userName,
				OTicket:        150,
				FShare:         75,
				DeleteTime:     3600,
				DefaultProject: "",
			}

			// Modify the user configuration
			err = qc.ModifyUser(userName, newUserConfig)
			Expect(err).To(BeNil())

			// Show the specific user configuration and verify its details
			retrievedUserConfig, err = qc.ShowUser(userName)
			Expect(err).To(BeNil())
			qconf.SetDefaultUserValues(&newUserConfig)
			Expect(retrievedUserConfig).To(Equal(newUserConfig))

			// Delete the user
			err = qc.DeleteUser([]string{userName})
			Expect(err).To(BeNil())

			// Show all users, should no longer contain the deleted user
			users, err = qc.ShowUsers()
			Expect(err).To(BeNil())
			Expect(users).NotTo(BeNil())
			Expect(users).NotTo(ContainElement(userName))

			// Show the specific user configuration, should return an error
			_, err = qc.ShowUser(userName)
			Expect(err).NotTo(BeNil())

			qc.DeleteProject([]string{"defaultP"})
		})
	})

	Context("Attribute Modification", func() {

		It("should modify attributes for specified objects", func() {
			objName := "queue"
			attrName := "slots"
			val := "10"
			objIDList := "all.q"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Set 10 slots for all all.q queue instances
			err = qc.ModifyAttribute(objName, attrName, val, objIDList)
			Expect(err).To(BeNil())
		})

		It("should delete attributes from specified objects", func() {
			objName := "queue"
			attrName := "qtype"
			val := "INTERACTIVE"
			objIDList := "all.q"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Don't allow interactive jobs in all.q queue instances'
			err = qc.DeleteAttribute(objName, attrName, val, objIDList)
			Expect(err).To(BeNil())

			// Incosistency in the CLI. The attribute value is completely
			// replaced. This should be named SetAttribute (sattr).
			err = qc.AddAttribute(objName, attrName, "INTERACTIVE BATCH", objIDList)
			Expect(err).To(BeNil())

			// @TODO missing on CLI
			//attr = qc.ShowAttribute(objName, attrName, objIDList)
		})
	})

	Context("Scheduler Configuration", func() {

		var backup *qconf.SchedulerConfig

		var qc *qconf.CommandLineQConf
		var err error

		BeforeEach(func() {
			qc, err = qconf.NewCommandLineQConf(
				qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())
			backup, err = qc.ShowSchedulerConfiguration()
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			err = qc.ModifySchedulerConfig(*backup)
			Expect(err).To(BeNil())
		})

		It("should show and modify the scheduler configuration", func() {

			// Show the current scheduler configuration
			schedulerConfig, err := qc.ShowSchedulerConfiguration()
			Expect(err).To(BeNil())

			// Modify the scheduler configuration
			schedulerConfig.MaxUJobs = 100
			schedulerConfig.MaxReservation = 1000
			schedulerConfig.MaxFunctionalJobsToSchedule = 10000
			schedulerConfig.DefaultDuration = "01:00:00"

			// Update HalflifeDecayList with values representing decay rates
			schedulerConfig.HalflifeDecayList = []string{"cpu=0.5", "io=0.75", "mem=0.9"}

			// Update JobLoadAdjustments with some test values
			schedulerConfig.JobLoadAdjustments = []string{"np_load_avg=1.00", "mem_free=1.00"}

			// Update UsageWeightList with test values
			schedulerConfig.UsageWeightList = []string{"cpu=0.800000", "mem=0.100000", "io=100000.000000"}

			schedulerConfig.Params = []string{"MONITOR=1", "PROFILE=1"}

			// Update policy hierarchy
			schedulerConfig.PolicyHierarchy = "OFS"

			err = qc.ModifySchedulerConfig(*schedulerConfig)
			Expect(err).To(BeNil())

			// Show the modified scheduler configuration
			modifiedSchedulerConfig, err := qc.ShowSchedulerConfiguration()
			Expect(err).To(BeNil())
			Expect(modifiedSchedulerConfig).To(Equal(schedulerConfig))

			// revert to original configuration
			err = qc.ModifySchedulerConfig(*backup)
			Expect(err).To(BeNil())

			revertedConfig, err := qc.ShowSchedulerConfiguration()
			Expect(err).To(BeNil())
			Expect(revertedConfig).To(Equal(backup))

		})

		It("should modify with an empty configuration", func() {
			err = qc.ModifySchedulerConfig(qconf.SchedulerConfig{})
			Expect(err).To(BeNil())
		})

	})

	Context("Share Tree Modification", func() {

		var qc *qconf.CommandLineQConf
		var err error

		var originalShareTree string

		BeforeEach(func() {
			qc, err = qconf.NewCommandLineQConf(
				qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())
			originalShareTree, _ = qc.ShowShareTree()

			// Add projects P1 and P2
			qc.AddProject(qconf.ProjectConfig{Name: "P10"})
			qc.AddProject(qconf.ProjectConfig{Name: "P20"})
		})

		AfterEach(func() {
			err = qc.ModifyShareTree(originalShareTree)
			Expect(err).To(BeNil())
			err = qc.DeleteProject([]string{"P10", "P20"})
			Expect(err).To(BeNil())
		})

		It("should modify the share tree", func() {
			shareTreeConfig := `id=0
name=Root
type=0
shares=1
childnodes=1,2,3
id=1
name=default
type=0
shares=10
childnodes=NONE
id=2
name=P20
type=1
shares=11
childnodes=NONE
id=3
name=P10
type=1
shares=11
childnodes=NONE
`
			err = qc.ModifyShareTree(shareTreeConfig)
			Expect(err).To(BeNil())

			modifiedShareTree, err := qc.ShowShareTree()
			Expect(err).To(BeNil())
			Expect(modifiedShareTree).To(Equal(shareTreeConfig))

			nodes, err := qc.ShowShareTreeNodes(nil)
			Expect(err).To(BeNil())
			Expect(nodes).To(ContainElement(
				qconf.ShareTreeNode{
					Node: "/P10", Share: 11}))

			err = qc.DeleteShareTreeNodes([]string{"P10"})
			Expect(err).To(BeNil())

			nodes, err = qc.ShowShareTreeNodes(nil)
			Expect(err).To(BeNil())
			Expect(nodes).NotTo(ContainElement(
				qconf.ShareTreeNode{
					Node: "/P10", Share: 11}))

			err = qc.AddShareTreeNode(
				qconf.ShareTreeNode{
					Node:  "/P10",
					Share: 11,
				})
			Expect(err).To(BeNil())

			err = qc.ModifyShareTreeNodes(
				[]qconf.ShareTreeNode{
					{Node: "/P10", Share: 12},
				})
			Expect(err).To(BeNil())

			nodes, err = qc.ShowShareTreeNodes(nil)
			Expect(err).To(BeNil())
			Expect(nodes).To(ContainElement(
				qconf.ShareTreeNode{
					Node: "/P10", Share: 12}))

			err = qc.DeleteShareTree()
			Expect(err).To(BeNil())

			_, err = qc.ShowShareTree()
			Expect(err).To(HaveOccurred())
		})

		It("should clear the share tree usage", func() {
			err = qc.ClearShareTreeUsage()
			Expect(err).To(BeNil())
		})

	})

	/*
		Context("Cluster Control Operations", func() {

			It("should clear job usage records correctly", func() {
				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				err = qc.ClearUsage()
				Expect(err).To(BeNil())
			})

			It("should clean queue job records correctly", func() {
				queueName := "all.q"

				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				// Clean the queue by its name
				err = qc.CleanQueue([]string{queueName})
				Expect(err).To(BeNil())
			})

			It("should shutdown execution daemons on specific hosts", func() {
				hostName := "localhost"

				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				// Shutdown the execution daemon on the specific host
				err = qc.ShutdownExecDaemons([]string{hostName})
				Expect(err).To(BeNil())
			})

			It("should shutdown the master daemon", func() {
				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				// Shutdown the master daemon
				err = qc.ShutdownMasterDaemon()
				Expect(err).To(BeNil())
			})

			It("should shutdown the scheduling daemon", func() {
				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				// Shutdown the scheduling daemon
				err = qc.ShutdownSchedulingDaemon()
				Expect(err).To(BeNil())
			})

			It("should kill specific event clients by their IDs", func() {
				eventID := "1234"

				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				// Kill the event client by its ID
				err = qc.KillEventClient([]string{eventID})
				Expect(err).To(BeNil())
			})

			It("should kill specific qmaster threads by their names", func() {
				threadName := "test-thread"

				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				// Kill the qmaster thread by its name
				err = qc.KillQmasterThread(threadName)
				Expect(err).To(BeNil())
			})

		})


		Context("Checkpointing Interface Modification", func() {
			It("should modify checkpointing interfaces", func() {
				ckptName := "test-ckpt"
				newCkptConfig := qconf.CkptInterfaceConfig{
					Name:           ckptName,
					Interface:      "test-interface",
					CleanCommand:   "/path/to/new_clean_command",
					CheckpointCmd:  "/path/to/new_checkpoint_cmd",
					MigrCommand:    "/path/to/new_migr_command",
					RestartCommand: "/path/to/new_restart_command",
					CkptDir:        "/path/to/new_ckpt_dir",
					When:           "sxr",
					Signal:         "signum",
				}

				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				// Modify the checkpointing interface
				err = qc.ModifyCkptInterface(ckptName, newCkptConfig)
				Expect(err).To(BeNil())
			})
		})

		Context("Host Configuration Modification", func() {
			It("should modify host configurations", func() {
				hostName := "test-host"
				newHostConfig := qconf.HostConfiguration{
					Name:   hostName,
					Mailer: "/new/mailer",
					Xterm:  "/new/xterm",
				}

				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				// Modify the host configuration
				err = qc.ModifyHostConfiguration(hostName, newHostConfig)
				Expect(err).To(BeNil())
			})
		})

		Context("Execution Host Modification", func() {
			It("should modify execution hosts", func() {
				hostName := "test-exec-host"
				newExecHostConfig := qconf.HostExecConfig{
					Hostname:    hostName,
					LoadScaling: "1.0",
				}

				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				// Modify the execution host configuration
				err = qc.ModifyExecHost(hostName, newExecHostConfig)
				Expect(err).To(BeNil())
			})
		})

		Context("Host Group Modification", func() {
			It("should modify host groups", func() {
				hostGroupName := "test-host-group"
				newHostGroupConfig := qconf.HostGroupConfig{
					GroupName: hostGroupName,
					Hostlist:  "newhost1 newhost2",
				}

				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				// Modify the host group configuration
				err = qc.ModifyHostGroup(hostGroupName, newHostGroupConfig)
				Expect(err).To(BeNil())
			})
		})

		Context("Parallel Environment Modification", func() {

			It("should modify parallel environments", func() {
				peName := "test-pe"
				newPeConfig := qconf.ParallelEnvironmentConfig{
					PeName:            peName,
					Slots:             200,
					UserLists:         "new_user_list",
					XUserLists:        "new_xuser_list",
					StartProcArgs:     "/new/start_proc",
					StopProcArgs:      "/new/stop_proc",
					AllocationRule:    "$pe_slots",
					ControlSlaves:     false,
					JobIsFirstTask:    false,
					UrgencySlots:      "200",
					AccountingSummary: false,
				}

				qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
				Expect(err).To(BeNil())

				// Modify the parallel environment configuration
				err = qc.ModifyParallelEnvironment(peName, newPeConfig)
				Expect(err).To(BeNil())
			})

		})
	*/

})
