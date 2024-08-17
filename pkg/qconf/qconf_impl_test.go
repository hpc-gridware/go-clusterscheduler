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

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf"
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
			Name:           interfaceName,
			Interface:      "userdefined",
			CleanCommand:   "/path/to/clean_command",
			CheckpointCmd:  "/path/to/checkpoint_cmd",
			MigrCommand:    "/path/to/migr_command",
			RestartCommand: "/path/to/restart_command",
			CkptDir:        "/path/to/ckpt_dir",
			When:           "xmr",
			Signal:         "usr2",
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

		It("should show and modify the global configuration", func() {
			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Retrieve the current global configuration
			globalConfig, err := qc.ShowGlobalConfiguration()
			Expect(err).To(BeNil())
			Expect(globalConfig).NotTo(BeNil())

			// Modify the global configuration for the test
			modifiedConfig := globalConfig
			modifiedConfig.ExecdSpoolDir = "/new/spool/dir"
			modifiedConfig.AutoUserDeleteTime = 3600
			modifiedConfig.AdministratorMail = "admin@example.com"

			// Apply the modified configuration
			err = qc.ModifyGlobalConfig(modifiedConfig)
			Expect(err).To(BeNil())

			// Verify that the global configuration was correctly modified
			retrievedConfig, err := qc.ShowGlobalConfiguration()
			Expect(err).To(BeNil())
			Expect(retrievedConfig).To(Equal(modifiedConfig))
		})

		It("should handle bool and int fields correctly when modifying global configuration", func() {
			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Retrieve the current global configuration
			globalConfig, err := qc.ShowGlobalConfiguration()
			Expect(err).To(BeNil())
			Expect(globalConfig).NotTo(BeNil())

			// Modify the global configuration with emphasis on bool and int fields
			modifiedConfig := globalConfig
			modifiedConfig.EnforceProject = !globalConfig.EnforceProject
			modifiedConfig.MaxJobs = globalConfig.MaxJobs + 1

			// Apply the modified configuration
			err = qc.ModifyGlobalConfig(modifiedConfig)
			Expect(err).To(BeNil())

			// Verify that the bool and int fields are modified correctly
			retrievedConfig, err := qc.ShowGlobalConfiguration()
			Expect(err).To(BeNil())
			Expect(retrievedConfig.EnforceProject).To(Equal(modifiedConfig.EnforceProject))
			Expect(retrievedConfig.MaxJobs).To(Equal(modifiedConfig.MaxJobs))
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

		It("should show, add, list, and delete execution hosts", func() {
			hostName := "exec-host-test"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all execution hosts, initially should not contain the new host
			hosts, err := qc.ShowExecHosts()
			Expect(err).To(BeNil())
			Expect(hosts).NotTo(BeNil())
			Expect(hosts).NotTo(ContainElement(hostName))

			eh, err := qc.ShowExecHost("master")
			Expect(err).To(BeNil())
			Expect(eh).NotTo(BeNil())

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
			eh.ComplexValues = "test_mem=1024"
			err = qc.ModifyExecHost("master", eh)
			Expect(err).To(BeNil())

			eh, err = qc.ShowExecHost("master")
			Expect(err).To(BeNil())
			Expect(eh).NotTo(BeNil())

			Expect(eh.ComplexValues).To(Equal("test_mem=1024"))

			err = qc.DeleteAttribute("exechost", "complex_values", "test_mem", "master")
			Expect(err).To(BeNil())

			eh, err = qc.ShowExecHost("master")
			Expect(err).To(BeNil())
			Expect(eh).NotTo(BeNil())

			Expect(eh.ComplexValues).To(Equal("NONE"))

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

		It("should show, add, list, and delete host groups", func() {
			groupName := "@host-group-test"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all host groups, initially should not contain the new group
			groups, err := qc.ShowHostGroups()
			Expect(err).To(BeNil())
			Expect(groups).NotTo(BeNil())
			Expect(groups).NotTo(ContainElement(groupName))

			// Define a new host group configuration
			hostGroupConfig := qconf.HostGroupConfig{
				Name:     groupName,
				Hostlist: "master",
			}

			// Add the new host group
			err = qc.AddHostGroup(hostGroupConfig)
			Expect(err).To(BeNil())

			// Show all host groups, now should contain the new group
			groups, err = qc.ShowHostGroups()
			Expect(err).To(BeNil())
			Expect(groups).NotTo(BeNil())
			Expect(groups).To(ContainElement(groupName))

			// Show the specific host group configuration and verify its details
			retrievedHostGroupConfig, err := qc.ShowHostGroup(groupName)
			Expect(err).To(BeNil())
			Expect(retrievedHostGroupConfig).To(Equal(hostGroupConfig))

			retrievedHostGroupConfig.Hostlist = ""
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
				UserLists:         "arusers defaultdepartment",
				XUserLists:        "deadlineusers",
				StartProcArgs:     "/start/procedure",
				StopProcArgs:      "/stop/procedure",
				AllocationRule:    "$pe_slots",
				ControlSlaves:     true,
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
				ACL:     "",
				XACL:    "",
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
		It("should show, add, list, and delete cluster queues", func() {
			queueName := "cluster-queue-test"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

			// Show all cluster queues, initially should not contain the new queue
			queues, err := qc.ShowClusterQueues()
			Expect(err).To(BeNil())
			Expect(queues).NotTo(BeNil())
			Expect(queues).NotTo(ContainElement(queueName))

			// Define a new cluster queue configuration
			queueConfig := qconf.ClusterQueueConfig{
				Name:     queueName,
				HostList: "@allhosts",
				SeqNo:    77,
				Priority: 0,
				Slots:    10,
				// Add other necessary fields here...
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
			qconf.SetDefaultQueueValues(&queueConfig)
			Expect(retrievedQueueConfig).To(Equal(queueConfig))

			newQueueConfig := qconf.ClusterQueueConfig{
				Name:           queueName,
				HostList:       "@allhosts",
				SeqNo:          99,
				LoadThresholds: "np_load_avg=1.75",
				Slots:          50,
				MinCpuInterval: "00:01:00",
				QType:          "BATCH INTERACTIVE",
				Prolog:         "/new/prolog",
				Epilog:         "/new/epilog",
				InitialState:   "disabled",
			}
			err = qc.ModifyClusterQueue(queueName, newQueueConfig)
			Expect(err).To(BeNil())

			// Show the specific cluster queue configuration and verify its details
			retrievedQueueConfig, err = qc.ShowClusterQueue(queueName)

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

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
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

		It("should show, add users to, list, and delete from user set lists", func() {
			listName := "testlist"
			user := "root"

			qc, err := qconf.NewCommandLineQConf(qconf.CommandLineQConfConfig{Executable: "qconf"})
			Expect(err).To(BeNil())

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

			// Show all user set lists, now should contain the new list
			lists, err = qc.ShowUserSetLists()
			Expect(err).To(BeNil())
			Expect(lists).NotTo(BeNil())
			Expect(lists).To(ContainElement(listName))

			// Show the specific user set list configuration and verify its details
			retrievedUserSetListConfig, err := qc.ShowUserSetList(listName)
			Expect(err).To(BeNil())
			userSetListConfig.Entries = user
			Expect(retrievedUserSetListConfig).To(Equal(userSetListConfig))

			// Delete user from the user set list
			err = qc.DeleteUserFromUserSetList(user, listName)
			Expect(err).To(BeNil())

			retrievedUserSetListConfig, err = qc.ShowUserSetList(listName)
			Expect(err).To(BeNil())
			Expect(retrievedUserSetListConfig.Entries).To(Equal("NONE"))

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

			// missing on CLI
			//attr = qc.ShowAttribute(objName, attrName, objIDList)
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
