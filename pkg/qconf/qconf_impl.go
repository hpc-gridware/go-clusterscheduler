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

package qconf

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type CommandLineQConf struct {
	executable string
}

// NewCommandLineQConf creates a new instance of CommandLineQConf.
func NewCommandLineQConf(executable string) (*CommandLineQConf, error) {
	return &CommandLineQConf{executable: executable}, nil
}

// RunCommand executes the qconf command with the specified arguments.
func (c *CommandLineQConf) RunCommand(args ...string) (string, error) {
	cmd := exec.Command(c.executable, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("failed to run command (%s): %v",
			out.String(), err)
	}
	return out.String(), err
}

func GetEnvInt(env string) (int, error) {
	v, exists := os.LookupEnv(env)
	if !exists {
		return 0, fmt.Errorf("environment variable %s not set",
			env)
	}
	return strconv.Atoi(v)
}

func GetEnvironment() (ClusterEnviornment, error) {
	clusterEnviornment := ClusterEnviornment{}
	clusterEnviornment.Name = os.Getenv("SGE_CLUSTER_NAME")
	clusterEnviornment.Root = os.Getenv("SGE_ROOT")
	clusterEnviornment.Cell = os.Getenv("SGE_CELL")
	clusterEnviornment.QmasterPort, _ = GetEnvInt("SGE_QMASTER_PORT")
	clusterEnviornment.ExecdPort, _ = GetEnvInt("SGE_EXECD_PORT")
	return clusterEnviornment, nil
}

func (c *CommandLineQConf) ReadClusterConfiguration() (ClusterConfig, error) {
	cc := ClusterConfig{}

	// general settings which defines the environment
	var err error
	cc.ClusterEnviornment, err = GetEnvironment()
	if err != nil {
		return cc, fmt.Errorf("failed to read cluster environment: %v", err)
	}

	cc.GlobalConfig, err = c.ShowGlobalConfiguration()
	if err != nil {
		return cc, fmt.Errorf("failed to read global config: %v", err)
	}

	hostConfigs, err := c.ShowHostConfigurations()
	if err != nil {
		return cc, fmt.Errorf("failed to read host configs: %v", err)
	}
	for _, host := range hostConfigs {
		hc, err := c.ShowHostConfiguration(host)
		if err != nil {
			return cc, fmt.Errorf("failed to read host config: %v", err)
		}
		cc.HostConfigurations = append(cc.HostConfigurations, hc)
	}

	projectNames, err := c.ShowProjects()
	if err != nil {
		return cc, fmt.Errorf("failed to read projects: %v", err)
	}
	for _, projectName := range projectNames {
		pc, err := c.ShowProject(projectName)
		if err != nil {
			return cc, fmt.Errorf("failed to read project: %v", err)
		}
		cc.Projects = append(cc.Projects, pc)
	}

	// Read Calendars
	calendars, err := c.ShowCalendars()
	if err != nil {
		return cc, fmt.Errorf("failed to read calendars: %v", err)
	}
	for _, calendar := range calendars {
		ccal, err := c.ShowCalendar(calendar)
		if err != nil {
			return cc, fmt.Errorf("failed to read calendar: %v", err)
		}
		cc.Calendars = append(cc.Calendars, ccal)
	}

	// Read Complex Entries
	cc.ComplexEntries, err = c.ShowAllComplexes()
	if err != nil {
		return cc, fmt.Errorf("failed to read complex entries: %v", err)
	}

	// Read Ckpt Interfaces
	ckptInterfaces, err := c.ShowCkptInterfaces()
	if err != nil {
		return cc, fmt.Errorf("failed to read ckpt interfaces: %v", err)
	}
	for _, ckptInterface := range ckptInterfaces {
		ci, err := c.ShowCkptInterface(ckptInterface)
		if err != nil {
			return cc, fmt.Errorf("failed to read ckpt interface: %v", err)
		}
		cc.CkptInterfaces = append(cc.CkptInterfaces, ci)
	}

	// Read Exec Hosts
	execHosts, err := c.ShowExecHosts()
	if err != nil {
		return cc, fmt.Errorf("failed to read exec hosts: %v", err)
	}
	for _, execHost := range execHosts {
		eh, err := c.ShowExecHost(execHost)
		if err != nil {
			return cc, fmt.Errorf("failed to read exec host: %v", err)
		}
		cc.ExecHosts = append(cc.ExecHosts, eh)
	}

	// Read Admin Hosts
	adminHosts, err := c.ShowAdminHosts()
	if err != nil {
		return cc, fmt.Errorf("failed to read admin hosts: %v", err)
	}
	cc.AdminHosts = adminHosts

	// Read Host Groups
	hostGroups, err := c.ShowHostGroups()
	if err != nil {
		return cc, fmt.Errorf("failed to read host groups: %v", err)
	}
	for _, hostGroup := range hostGroups {
		hg, err := c.ShowHostGroup(hostGroup)
		if err != nil {
			return cc, fmt.Errorf("failed to read host group: %v", err)
		}
		cc.HostGroups = append(cc.HostGroups, hg)
	}

	// Read Resource Quota Sets
	resourceQuotaSets, err := c.ShowResourceQuotaSets()
	if err != nil {
		return cc, fmt.Errorf("failed to read resource quota sets: %v", err)
	}
	for _, resourceQuotaSet := range resourceQuotaSets {
		rqs, err := c.ShowResourceQuotaSet(resourceQuotaSet)
		if err != nil {
			return cc, fmt.Errorf("failed to read resource quota set: %v", err)
		}
		cc.ResourceQuotaSets = append(cc.ResourceQuotaSets, rqs)
	}

	// Read Managers
	managers, err := c.ShowManagers()
	if err != nil {
		return cc, fmt.Errorf("failed to read managers: %v", err)
	}
	cc.Managers = managers

	// Read Operators
	operators, err := c.ShowOperators()
	if err != nil {
		return cc, fmt.Errorf("failed to read operators: %v", err)
	}
	cc.Operators = operators

	// Read Parallel Environments
	parallelEnvironments, err := c.ShowParallelEnvironments()
	if err != nil {
		return cc, fmt.Errorf("failed to read parallel environments: %v", err)
	}
	for _, parallelEnvironment := range parallelEnvironments {
		pe, err := c.ShowParallelEnvironment(parallelEnvironment)
		if err != nil {
			return cc, fmt.Errorf("failed to read parallel environment: %v", err)
		}
		cc.ParallelEnvironments = append(cc.ParallelEnvironments, pe)
	}

	// Read Users
	users, err := c.ShowUsers()
	if err != nil {
		return cc, fmt.Errorf("failed to read users: %v", err)
	}
	for _, user := range users {
		u, err := c.ShowUser(user)
		if err != nil {
			return cc, fmt.Errorf("failed to read user: %v", err)
		}
		cc.Users = append(cc.Users, u)
	}

	// Read Cluster Queues
	clusterQueues, err := c.ShowClusterQueues()
	if err != nil {
		return cc, fmt.Errorf("failed to read cluster queues: %v", err)
	}
	for _, clusterQueue := range clusterQueues {
		cq, err := c.ShowClusterQueue(clusterQueue)
		if err != nil {
			return cc, fmt.Errorf("failed to read cluster queue: %v", err)
		}
		cc.ClusterQueues = append(cc.ClusterQueues, cq)
	}

	// Read User Set Lists
	userSetLists, err := c.ShowUserSetLists()
	if err != nil {
		return cc, fmt.Errorf("failed to read user set lists: %v", err)
	}
	for _, userSetList := range userSetLists {
		usl, err := c.ShowUserSetList(userSetList)
		if err != nil {
			return cc, fmt.Errorf("failed to read user set list: %v", err)
		}
		cc.UserSetLists = append(cc.UserSetLists, usl)
	}

	return cc, nil
}

// AddCalendar adds a new calendar.
func (c *CommandLineQConf) AddCalendar(cfg CalendarConfig) error {
	// Create file in tmp directory with calendar configuration.
	// Use the file as input to the qconf command.
	// Remove the file after the command completes.

	if cfg.CalendarName == "" {
		return fmt.Errorf("calendar name is required")
	}
	if cfg.Year == "" {
		cfg.Year = "NONE"
	}
	if cfg.Week == "" {
		cfg.Week = "NONE"
	}

	file, err := os.CreateTemp("", "calendar")
	if err != nil {
		return err
	}
	defer os.Remove("calendar")

	_, err = file.WriteString(fmt.Sprintf("calendar_name    %s\n", cfg.CalendarName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("year             %s\n", cfg.Year))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("week             %s\n", cfg.Week))
	if err != nil {
		return err
	}
	file.Close()
	_, err = c.RunCommand("-Acal", file.Name())
	if err != nil {
		return fmt.Errorf("failed to add calendar: %v", err)
	}
	return err
}

// DeleteCalendar deletes a calendar.
func (c *CommandLineQConf) DeleteCalendar(calendarName string) error {
	_, err := c.RunCommand("-dcal", calendarName)
	return err
}

// ShowCalendar shows the specified calendar.
func (c *CommandLineQConf) ShowCalendar(calendarName string) (CalendarConfig, error) {
	out, err := c.RunCommand("-scal", calendarName)
	if err != nil {
		return CalendarConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := CalendarConfig{CalendarName: calendarName}
	for _, line := range lines {
		if strings.HasPrefix(line, "year") {
			cfg.Year = strings.TrimSpace(strings.Fields(line)[1])
		} else if strings.HasPrefix(line, "week") {
			cfg.Week = strings.TrimSpace(strings.Fields(line)[1])
		} else if strings.HasPrefix(line, "calendar_name") {
			cfg.CalendarName = strings.TrimSpace(strings.Fields(line)[1])
		}
	}
	return cfg, nil
}

// ShowCalendars shows all calendars.
func (c *CommandLineQConf) ShowCalendars() ([]string, error) {
	output, err := c.RunCommand("-scall")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	// each line is a calender name
	return splitWithoutEmptyLines(output, "\n"), nil
}

func SetDefaultComplexEntryValues(c *ComplexEntryConfig) {
	if c.Shortcut == "" {
		c.Shortcut = c.Name
	}
	if c.Requestable == "" {
		c.Requestable = "NO"
	}
	if c.Consumable == "" {
		c.Consumable = "NO"
	}
	if c.Default == "" {
		if c.Type == "STRING" {
			c.Default = "NONE"
		} else {
			c.Default = "0"
		}
	}
	if c.Relop == "" {
		c.Relop = "=="
	}
}

// AddComplexEntry adds a new complex entry.
func (c *CommandLineQConf) AddComplexEntry(e ComplexEntryConfig) error {
	if e.Name == "" {
		return fmt.Errorf("complex does not have a name")
	}
	if e.Type == "" {
		return fmt.Errorf("complex does not have a type")
	}
	SetDefaultComplexEntryValues(&e)
	file, err := createTempDirWithFileName(e.Name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	// Format complex entry configuration.
	_, err = file.WriteString(fmt.Sprintf("name        %s\n", e.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("shortcut    %s\n", e.Shortcut))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("type        %s\n", e.Type))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("relop       %s\n", e.Relop))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("requestable %s\n", e.Requestable))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("consumable  %s\n", e.Consumable))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("default     %s\n", e.Default))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("urgency     %d\n", e.Urgency))
	if err != nil {
		return err
	}
	file.Close()

	// CS-463
	out, err := c.RunCommand("-Ace", file.Name())
	if err != nil {
		if strings.Contains(out, "added") {
			// ignore exit code 1
			return nil
		}
		return fmt.Errorf("failed to add complex entry (%s): %v",
			out, err)
	}
	return nil
}

// DeleteComplexEntry deletes a complex entry.
func (c *CommandLineQConf) DeleteComplexEntry(entryName string) error {
	_, err := c.RunCommand("-dce", entryName)
	return err
}

// ShowComplexEntry shows the specified complex entry.
func (c *CommandLineQConf) ShowComplexEntry(entryName string) (ComplexEntryConfig, error) {
	out, err := c.RunCommand("-sce", entryName)
	if err != nil {
		return ComplexEntryConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := ComplexEntryConfig{Name: entryName}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "name":
			cfg.Name = fields[1]
		case "shortcut":
			cfg.Shortcut = fields[1]
		case "type":
			cfg.Type = fields[1]
		case "relop":
			cfg.Relop = fields[1]
		case "requestable":
			cfg.Requestable = fields[1]
		case "consumable":
			cfg.Consumable = fields[1]
		case "default":
			// Use strings.Join to handle additional spaces
			cfg.Default = fields[1]
			if len(fields) > 2 {
				cfg.Default = strings.Join(fields[1:], " ")
			}
		case "urgency":
			urgency, err := strconv.Atoi(fields[1])
			if err != nil {
				return ComplexEntryConfig{},
					fmt.Errorf("invalid urgency value: %v", err)
			}
			cfg.Urgency = urgency
		}
	}
	return cfg, nil
}

// ShowComplexEntries shows all complex entries.
func (c *CommandLineQConf) ShowComplexEntries() ([]string, error) {
	output, err := c.RunCommand("-scel")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	// each line is a complex entry name
	return splitWithoutEmptyLines(output, "\n"), nil
}

// ShowAllComplexes shows all complexes. It makes one call to
// the backend is much more efficient than calling ShowComplexEntries
// and ShowComplexEntry for each entry.
func (c *CommandLineQConf) ShowAllComplexes() ([]ComplexEntryConfig, error) {
	output, err := c.RunCommand("-sc")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(output, "\n")
	var complexes []ComplexEntryConfig
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 8 {
			continue
		}
		urgency, err := strconv.Atoi(fields[7])
		if err != nil {
			return nil, fmt.Errorf("Unexpected failured when parsing: %v", err)
		}
		complexes = append(complexes,
			ComplexEntryConfig{
				Name:        fields[0],
				Shortcut:    fields[1],
				Type:        fields[2],
				Relop:       fields[3],
				Requestable: fields[4],
				Consumable:  fields[5],
				Default:     fields[6],
				Urgency:     urgency,
			})
	}

	return complexes, nil
}

// AddCkptInterface adds a new checkpointing interface.
func (c *CommandLineQConf) AddCkptInterface(cfg CkptInterfaceConfig) error {
	// Create a temporary file with the checkpointing interface configuration
	file, err := createTempDirWithFileName(cfg.Name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	// Write configuration to the temporary file
	// Inconsistency: unknown attribute name \"name\"
	_, err = file.WriteString(fmt.Sprintf("ckpt_name           %s\n", cfg.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("interface      %s\n", cfg.Interface))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("ckpt_command  %s\n", cfg.CheckpointCmd))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("ckpt_dir      %s\n", cfg.CkptDir))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("migr_command    %s\n", cfg.MigrCommand))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("restart_command %s\n", cfg.RestartCommand))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("clean_command   %s\n", cfg.CleanCommand))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("signal        %s\n", cfg.Signal))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("when   %s\n", cfg.When))
	if err != nil {
		return err
	}
	file.Close()

	// Execute the qconf command
	_, err = c.RunCommand("-Ackpt", file.Name())
	if err != nil {
		return fmt.Errorf("failed to add checkpointing interface: %v", err)
	}
	return nil
}

// DeleteCkptInterface deletes a checkpointing interface.
func (c *CommandLineQConf) DeleteCkptInterface(interfaceName string) error {
	_, err := c.RunCommand("-dckpt", interfaceName)
	return err
}

// ShowCkptInterface shows the specified checkpointing interface.
func (c *CommandLineQConf) ShowCkptInterface(interfaceName string) (CkptInterfaceConfig, error) {
	out, err := c.RunCommand("-sckpt", interfaceName)
	if err != nil {
		return CkptInterfaceConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := CkptInterfaceConfig{Name: interfaceName}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "ckpt_name":
			cfg.Name = fields[1]
		case "interface":
			cfg.Interface = fields[1]
		case "clean_command":
			cfg.CleanCommand = fields[1]
		case "ckpt_command":
			cfg.CheckpointCmd = fields[1]
		case "migr_command":
			cfg.MigrCommand = fields[1]
		case "restart_command":
			cfg.RestartCommand = fields[1]
		case "ckpt_dir":
			cfg.CkptDir = fields[1]
		case "signal":
			cfg.Signal = fields[1]
		case "when":
			cfg.When = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		}
	}
	return cfg, nil
}

// ShowCkptInterfaces shows all checkpointing interfaces.
func (c *CommandLineQConf) ShowCkptInterfaces() ([]string, error) {
	output, err := c.RunCommand("-sckptl")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	// Each line is a checkpointing interface name
	return splitWithoutEmptyLines(output, "\n"), nil
}

// AddHostConfiguration adds a new host configuration.
func (c *CommandLineQConf) AddHostConfiguration(config HostConfiguration) error {
	if config.Name == "" {
		return fmt.Errorf("Hostname not set in host configuration")
	}

	file, err := createTempDirWithFileName(config.Name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("mailer %s\n", config.Mailer))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xterm %s\n", config.Xterm))
	if err != nil {
		return err
	}
	file.Close()

	out, err := c.RunCommand("-Aconf", file.Name())
	if err != nil {
		return fmt.Errorf("failed to add host configuration (%s): %v",
			out, err)
	}
	return nil
}

// DeleteHostConfiguration deletes a host configuration.
func (c *CommandLineQConf) DeleteHostConfiguration(configName string) error {
	_, err := c.RunCommand("-dconf", configName)
	return err
}

// ShowHostConfiguration shows the specified host configuration.
func (c *CommandLineQConf) ShowHostConfiguration(hostName string) (HostConfiguration, error) {
	out, err := c.RunCommand("-sconf", hostName)
	if err != nil {
		return HostConfiguration{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := HostConfiguration{
		Name: hostName,
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "mailer":
			cfg.Mailer = fields[1]
		case "xterm":
			cfg.Xterm = fields[1]
		}
	}
	return cfg, nil
}

// parseMultiLineValue parses a multi-line value from the output.
// This is tricky because the output is not structured and the values can be
// split over multiple lines.
// The input is an array of all lines, the current index, the current line,
// and the fields of the current line. fields[0] is the detected key (like "reporting_params").
// The function returns the value and a boolean indicating if the value is multi-line.
//
// Example:
// ...
// qmaster_params               none
// execd_params                 none
//
//	reporting_params             accounting=true reporting=false finished_jobs=0 \
//		  test=blub test=bla
//
// ...
// lines is the array of all lines
// i is the line number with "reporting_params"
// The rule is that each non-multi-line output does not have a "  " prefix.
func ParseMultiLineValue(lines []string, i int) (string, bool) {
	line := lines[i]
	fields := strings.Fields(line)
	value := strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
	if strings.HasSuffix(value, "\\") {
		// multi-line value
		value = strings.TrimSuffix(value, "\\")
		for i, line := range lines {
			fds := strings.Fields(line)
			if len(fds) == 0 {
				continue
			}
			// find key like "reporting_params"
			if fds[0] == fields[0] {
				// multi-line values are indented by spaces, find all remaining lines
				for j := i + 1; j < len(lines) && strings.HasPrefix(lines[j], "  "); j++ {
					// Now the question is if we do at " " or "," or other
					// separators? We expect that the line ends with a separator.
					value += "" + strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(lines[j]), "\\"))
				}
			}
		}
		return value, true
	}
	return value, false
}

// ShowGlobalConfiguration shows the global configuration.
func (c *CommandLineQConf) ShowGlobalConfiguration() (GlobalConfig, error) {
	out, err := c.RunCommand("-sconf", "global")
	if err != nil {
		return GlobalConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := GlobalConfig{}
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		// Continue mapping fields as per GlobalConfig struct
		case "execd_spool_dir":
			cfg.ExecdSpoolDir = fields[1]
		case "mailer":
			cfg.Mailer = fields[1]
		case "xterm":
			cfg.Xterm = fields[1]
		case "load_sensor":
			cfg.LoadSensor = fields[1]
		case "prolog":
			cfg.Prolog = fields[1]
		case "epilog":
			cfg.Epilog = fields[1]
		case "shell_start_mode":
			cfg.ShellStartMode = fields[1]
		case "login_shells":
			cfg.LoginShells, _ = ParseMultiLineValue(lines, i)
		case "min_uid":
			// Assuming the value can be converted to an integer
			cfg.MinUID, _ = strconv.Atoi(fields[1])
		case "min_gid":
			cfg.MinGID, _ = strconv.Atoi(fields[1])
		case "user_lists":
			cfg.UserLists, _ = ParseMultiLineValue(lines, i)
		case "xuser_lists":
			cfg.XUserLists, _ = ParseMultiLineValue(lines, i)
		case "projects":
			cfg.Projects, _ = ParseMultiLineValue(lines, i)
		case "xprojects":
			cfg.XProjects, _ = ParseMultiLineValue(lines, i)
		case "enforce_project":
			cfg.EnforceProject, _ = strconv.ParseBool(fields[1])
		case "enforce_user":
			cfg.EnforceUser = fields[1]
		case "load_report_time":
			cfg.LoadReportTime = fields[1]
		case "max_unheard":
			cfg.MaxUnheard = fields[1]
		case "reschedule_unknown":
			cfg.RescheduleUnknown = fields[1]
		case "loglevel":
			cfg.LogLevel = fields[1]
		case "administrator_mail":
			cfg.AdministratorMail = fields[1]
		case "set_token_cmd":
			cfg.SetTokenCmd = fields[1]
		case "pag_cmd":
			cfg.PagCmd = fields[1]
		case "token_extend_time":
			cfg.TokenExtendTime = fields[1]
		case "shepherd_cmd":
			cfg.ShepherdCmd = fields[1]
		case "qmaster_params":
			cfg.QmasterParams = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		case "execd_params":
			cfg.ExecdParams = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		case "reporting_params":
			cfg.ReportingParams, _ = ParseMultiLineValue(lines, i)
		case "finished_jobs":
			cfg.FinishedJobs, _ = strconv.Atoi(fields[1])
		case "gid_range":
			cfg.GidRange = fields[1]
		case "qlogin_command":
			cfg.QloginCommand = fields[1]
		case "qlogin_daemon":
			cfg.QloginDaemon = fields[1]
		case "rlogin_command":
			cfg.RloginCommand = fields[1]
		case "rlogin_daemon":
			cfg.RloginDaemon = fields[1]
		case "rsh_command":
			cfg.RshCommand = fields[1]
		case "rsh_daemon":
			cfg.RshDaemon = fields[1]
		case "max_aj_instances":
			cfg.MaxAJInstances, _ = strconv.Atoi(fields[1])
		case "max_aj_tasks":
			cfg.MaxAJTasks, _ = strconv.Atoi(fields[1])
		case "max_u_jobs":
			cfg.MaxUJobs, _ = strconv.Atoi(fields[1])
		case "max_jobs":
			cfg.MaxJobs, _ = strconv.Atoi(fields[1])
		case "max_advance_reservations":
			cfg.MaxAdvanceReservations, _ = strconv.Atoi(fields[1])
		case "auto_user_oticket":
			cfg.AutoUserOTicket, _ = strconv.Atoi(fields[1])
		case "auto_user_fshare":
			cfg.AutoUserFshare, _ = strconv.Atoi(fields[1])
		case "auto_user_default_project":
			cfg.AutoUserDefaultProject = fields[1]
		case "auto_user_delete_time":
			cfg.AutoUserDeleteTime, _ = strconv.Atoi(fields[1])
		case "delegated_file_staging":
			cfg.DelegatedFileStaging, _ = strconv.ParseBool(fields[1])
		case "reprioritize":
			cfg.Reprioritize, _ = strconv.Atoi(fields[1])
		case "jsv_url":
			cfg.JsvURL = fields[1]
		case "jsv_allowed_mod":
			cfg.JsvAllowedMod = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		}
	}
	return cfg, nil
}

// ShowHostConfigurations shows all host configurations.
func (c *CommandLineQConf) ShowHostConfigurations() ([]string, error) {
	output, err := c.RunCommand("-sconfl")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

// AddExecHost adds a new execution host.
func (c *CommandLineQConf) AddExecHost(hostExecConfig HostExecConfig) error {
	file, err := createTempDirWithFileName(hostExecConfig.Hostname)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("hostname         %s\n", hostExecConfig.Hostname))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("load_scaling     %s\n", hostExecConfig.LoadScaling))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("complex_values   %s\n", hostExecConfig.ComplexValues))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("user_lists       %s\n", hostExecConfig.UserLists))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xuser_lists      %s\n", hostExecConfig.XUserLists))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("projects         %s\n", hostExecConfig.Projects))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xprojects        %s\n", hostExecConfig.XProjects))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("usage_scaling    %s\n", hostExecConfig.UsageScaling))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("report_variables %s\n", hostExecConfig.ReportVariables))
	if err != nil {
		return err
	}
	file.Close()

	out, err := c.RunCommand("-Ae", file.Name())
	if err != nil {
		return fmt.Errorf("failed to add exechost (%s): %v", out, err)
	}
	return nil
}

// DeleteExecHost deletes an execution host.
func (c *CommandLineQConf) DeleteExecHost(hostList string) error {
	_, err := c.RunCommand("-de", hostList)
	return err
}

// ShowExecHost shows the specified execution host.
func (c *CommandLineQConf) ShowExecHost(hostName string) (HostExecConfig, error) {
	out, err := c.RunCommand("-se", hostName)
	if err != nil {
		return HostExecConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := HostExecConfig{Hostname: hostName}
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "hostname":
			cfg.Hostname = fields[1]
		case "load_scaling":
			cfg.LoadScaling = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		case "complex_values":
			cfg.ComplexValues, _ = ParseMultiLineValue(lines, i)
		case "user_lists":
			cfg.UserLists, _ = ParseMultiLineValue(lines, i)
		case "xuser_lists":
			cfg.XUserLists, _ = ParseMultiLineValue(lines, i)
		case "projects":
			cfg.Projects, _ = ParseMultiLineValue(lines, i)
		case "xprojects":
			cfg.XProjects, _ = ParseMultiLineValue(lines, i)
		case "usage_scaling":
			cfg.UsageScaling, _ = ParseMultiLineValue(lines, i)
		case "report_variables":
			cfg.ReportVariables, _ = ParseMultiLineValue(lines, i)
		}
	}
	return cfg, nil
}

// ShowExecHosts shows all execution hosts.
func (c *CommandLineQConf) ShowExecHosts() ([]string, error) {
	output, err := c.RunCommand("-sel")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	// do not return empty strings
	return splitWithoutEmptyLines(output, "\n"), nil
}

func splitWithoutEmptyLines(content, sep string) []string {
	out := strings.Split(content, sep)
	if out == nil {
		return nil
	}
	var res []string
	for _, i := range out {
		if strings.TrimSpace(i) == "" {
			continue
		}
		res = append(res, strings.TrimSpace(i))
	}
	return res
}

// AddAdminHost adds a new administrative host.
func (c *CommandLineQConf) AddAdminHost(hosts []string) error {
	hostList := strings.Join(hosts, ",")
	_, err := c.RunCommand("-ah", hostList)
	if err != nil {
		return fmt.Errorf("failed to add adminhost: %v", err)
	}
	return nil
}

// DeleteAdminHost deletes an administrative host.
func (c *CommandLineQConf) DeleteAdminHost(hosts []string) error {
	hostList := strings.Join(hosts, ",")
	_, err := c.RunCommand("-dh", hostList)
	if err != nil {
		return fmt.Errorf("failed to delete adminhost: %v", err)
	}
	return nil
}

// ShowAdminHosts shows all administrative hosts.
func (c *CommandLineQConf) ShowAdminHosts() ([]string, error) {
	output, err := c.RunCommand("-sh")
	if err != nil {
		return nil, fmt.Errorf("failed to show adminhosts: %v", err)
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

// AddHostGroup adds a new host group.
func (c *CommandLineQConf) AddHostGroup(hostGroup HostGroupConfig) error {
	file, err := createTempDirWithFileName(hostGroup.GroupName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	if !strings.HasPrefix(hostGroup.GroupName, "@") {
		return fmt.Errorf("group name must start with '@'")
	}

	_, err = file.WriteString(fmt.Sprintf("group_name %s\n", hostGroup.GroupName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("hostlist %s\n", hostGroup.Hostlist))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Ahgrp", file.Name())
	return err
}

// DeleteHostGroup deletes a host group.
func (c *CommandLineQConf) DeleteHostGroup(groupName string) error {
	_, err := c.RunCommand("-dhgrp", groupName)
	return err
}

// ShowHostGroup shows the specified host group.
func (c *CommandLineQConf) ShowHostGroup(groupName string) (HostGroupConfig, error) {
	out, err := c.RunCommand("-shgrp", groupName)
	if err != nil {
		return HostGroupConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := HostGroupConfig{GroupName: groupName}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "group_name":
			cfg.GroupName = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		case "hostlist":
			cfg.Hostlist = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		}
	}
	return cfg, nil
}

// ShowHostGroups shows all host groups.
func (c *CommandLineQConf) ShowHostGroups() ([]string, error) {
	output, err := c.RunCommand("-shgrpl")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

// AddResourceQuotaSet adds a new resource quota set.
func (c *CommandLineQConf) AddResourceQuotaSet(rqs ResourceQuotaSetConfig) error {
	SetResourceQuotaSetDefaults(&rqs)
	file, err := createTempDirWithFileName(rqs.Name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))
	_, err = file.WriteString(fmt.Sprintf("{\n"))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("name         %s\n", rqs.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("description  %s\n", rqs.Description))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("enabled      %v\n", rqs.Enabled))
	if err != nil {
		return err
	}
	for _, limit := range rqs.Limits {
		_, err = file.WriteString(fmt.Sprintf("limit        %s\n", limit))
		if err != nil {
			return err
		}
	}
	_, err = file.WriteString(fmt.Sprintf("}\n"))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Arqs", file.Name())
	return err
}

// DeleteResourceQuotaSet deletes a resource quota set.
func (c *CommandLineQConf) DeleteResourceQuotaSet(rqsList string) error {
	_, err := c.RunCommand("-drqs", rqsList)
	return err
}

// ShowResourceQuotaSet shows the specified resource quota set.
func (c *CommandLineQConf) ShowResourceQuotaSet(rqsList string) (ResourceQuotaSetConfig, error) {
	out, err := c.RunCommand("-srqs", rqsList)
	if err != nil {
		return ResourceQuotaSetConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := ResourceQuotaSetConfig{Name: rqsList}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "name":
			cfg.Name = fields[1]
		case "description":
			cfg.Description = strings.TrimSpace(
				strings.TrimPrefix(line, fields[0]))
		case "enabled":
			cfg.Enabled, _ = strconv.ParseBool(fields[1])
		case "limit":
			cfg.Limits = append(cfg.Limits,
				strings.TrimSpace(strings.TrimPrefix(line, fields[0])))
		}
	}
	return cfg, nil
}

// ShowResourceQuotaSets shows all resource quota sets.
func (c *CommandLineQConf) ShowResourceQuotaSets() ([]string, error) {
	output, err := c.RunCommand("-srqsl")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

// AddUserToManagerList adds a list of users to the manager list.
func (c *CommandLineQConf) AddUserToManagerList(users []string) error {
	_, err := c.RunCommand("-am", strings.Join(users, ","))
	return err
}

// DeleteUserFromManagerList deletes a list of users from the manager list.
func (c *CommandLineQConf) DeleteUserFromManagerList(users []string) error {
	_, err := c.RunCommand("-dm", strings.Join(users, ","))
	return err
}

// ShowManagers shows the manager list.
func (c *CommandLineQConf) ShowManagers() ([]string, error) {
	output, err := c.RunCommand("-sm")
	if err != nil {
		return nil, err
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

// AddUserToOperatorList adds a list of users to the operator list.
func (c *CommandLineQConf) AddUserToOperatorList(users []string) error {
	_, err := c.RunCommand("-ao", strings.Join(users, ","))
	return err
}

// DeleteUserFromOperatorList deletes a list of users from the operator list.
func (c *CommandLineQConf) DeleteUserFromOperatorList(users []string) error {
	_, err := c.RunCommand("-do", strings.Join(users, ","))
	return err
}

// ShowOperators shows the operator list.
func (c *CommandLineQConf) ShowOperators() ([]string, error) {
	output, err := c.RunCommand("-so")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

func SetDefaultParallelEnvironmentValues(pe *ParallelEnvironmentConfig) {
	if pe.UserLists == "" {
		pe.UserLists = "NONE"
	}
	if pe.XUserLists == "" {
		pe.XUserLists = "NONE"
	}
	if pe.StartProcArgs == "" {
		pe.StartProcArgs = "/bin/true"
	}
	if pe.StopProcArgs == "" {
		pe.StopProcArgs = "/bin/true"
	}
	if pe.AllocationRule == "" {
		pe.AllocationRule = "$pe_slots"
	}
	if pe.UrgencySlots == "" {
		pe.UrgencySlots = "min"
	}
}

func MakeBoolCfg(v bool) string {
	if v {
		return "TRUE"
	}
	return "FALSE"
}

// AddParallelEnvironment adds a new parallel environment.
func (c *CommandLineQConf) AddParallelEnvironment(pe ParallelEnvironmentConfig) error {
	SetDefaultParallelEnvironmentValues(&pe)
	file, err := createTempDirWithFileName(pe.PeName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	err = writePE(file, pe)
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Ap", file.Name())
	return err
}

func writePE(file *os.File, pe ParallelEnvironmentConfig) error {
	_, err := file.WriteString(fmt.Sprintf("pe_name            %s\n", pe.PeName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("slots             %d\n", pe.Slots))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("user_lists         %s\n", pe.UserLists))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xuser_lists        %s\n", pe.XUserLists))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("start_proc_args     %s\n", pe.StartProcArgs))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("stop_proc_args      %s\n", pe.StopProcArgs))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("allocation_rule    %s\n", pe.AllocationRule))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("control_slaves     %s\n", MakeBoolCfg(pe.ControlSlaves)))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("job_is_first_task  %s\n", MakeBoolCfg(pe.JobIsFirstTask)))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("urgency_slots      %s\n", pe.UrgencySlots))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("accounting_summary %s\n", MakeBoolCfg(pe.AccountingSummary)))
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

// DeleteParallelEnvironment deletes a parallel environment.
func (c *CommandLineQConf) DeleteParallelEnvironment(peName string) error {
	_, err := c.RunCommand("-dp", peName)
	return err
}

// ShowParallelEnvironment shows the specified parallel environment.
func (c *CommandLineQConf) ShowParallelEnvironment(peName string) (ParallelEnvironmentConfig, error) {
	out, err := c.RunCommand("-sp", peName)
	if err != nil {
		return ParallelEnvironmentConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := ParallelEnvironmentConfig{PeName: peName}
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "pe_name":
			cfg.PeName = fields[1]
		case "slots":
			cfg.Slots, _ = strconv.Atoi(fields[1])
		case "user_lists":
			cfg.UserLists, _ = ParseMultiLineValue(lines, i)
		case "xuser_lists":
			cfg.XUserLists, _ = ParseMultiLineValue(lines, i)
		case "start_proc_args":
			cfg.StartProcArgs, _ = ParseMultiLineValue(lines, i)
		case "stop_proc_args":
			cfg.StopProcArgs, _ = ParseMultiLineValue(lines, i)
		case "allocation_rule":
			cfg.AllocationRule = fields[1]
		case "control_slaves":
			cfg.ControlSlaves, _ = strconv.ParseBool(fields[1])
		case "job_is_first_task":
			cfg.JobIsFirstTask, _ = strconv.ParseBool(fields[1])
		case "urgency_slots":
			cfg.UrgencySlots = fields[1]
		case "accounting_summary":
			cfg.AccountingSummary, _ = strconv.ParseBool(fields[1])
		}
	}
	return cfg, nil
}

// ShowParallelEnvironments shows all parallel environments.
func (c *CommandLineQConf) ShowParallelEnvironments() ([]string, error) {
	output, err := c.RunCommand("-spl")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

func SetDefaultProjectValues(project *ProjectConfig) {
	if project.ACL == "" {
		project.ACL = "NONE"
	}
	if project.XACL == "" {
		project.XACL = "NONE"
	}
}

// AddProject adds a project.
func (c *CommandLineQConf) AddProject(project ProjectConfig) error {
	SetDefaultProjectValues(&project)
	file, err := createTempDirWithFileName(project.Name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("name    %s\n", project.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("oticket %d\n", project.OTicket))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("fshare  %d\n", project.FShare))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("acl     %s\n", project.ACL))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xacl    %s\n", project.XACL))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Aprj", file.Name())
	return err
}

// DeleteProject deletes the specified projects.
func (c *CommandLineQConf) DeleteProject(projects []string) error {
	_, err := c.RunCommand("-dprj", strings.Join(projects, ","))
	return err
}

// ShowProject shows the specified project.
func (c *CommandLineQConf) ShowProject(projectName string) (ProjectConfig, error) {
	out, err := c.RunCommand("-sprj", projectName)
	if err != nil {
		return ProjectConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := ProjectConfig{Name: projectName}
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "name":
			cfg.Name = fields[1]
		case "oticket":
			cfg.OTicket, _ = strconv.Atoi(fields[1])
		case "fshare":
			cfg.FShare, _ = strconv.Atoi(fields[1])
		case "acl":
			cfg.ACL, _ = ParseMultiLineValue(lines, i)
		case "xacl":
			cfg.XACL, _ = ParseMultiLineValue(lines, i)
		}
	}
	return cfg, nil
}

// ShowProjects shows all projects.
func (c *CommandLineQConf) ShowProjects() ([]string, error) {
	output, err := c.RunCommand("-sprjl")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

func SetDefaultQueueValues(queue *ClusterQueueConfig) {
	if queue.QName == "" {
		queue.QName = "test"
	}
	if queue.HostList == "" {
		queue.HostList = "NONE"
	}
	if queue.SeqNo == 0 {
		queue.SeqNo = 0
	}
	if queue.LoadThresholds == "" {
		queue.LoadThresholds = "np_load_avg=1.75"
	}
	if queue.SuspendThresholds == "" {
		queue.SuspendThresholds = "NONE"
	}
	if queue.NSuspend == 0 {
		queue.NSuspend = 0
	}
	if queue.SuspendInterval == "" {
		queue.SuspendInterval = "00:05:00"
	}
	if queue.Priority == 0 {
		queue.Priority = 0
	}
	if queue.MinCpuInterval == "" {
		queue.MinCpuInterval = "00:05:00"
	}
	if queue.Processors == "" {
		queue.Processors = "UNDEFINED"
	}
	if queue.QType == "" {
		queue.QType = "BATCH INTERACTIVE"
	}
	if queue.CkptList == "" {
		queue.CkptList = "NONE"
	}
	if queue.PeList == "" {
		queue.PeList = "make"
	}
	if queue.Rerun == false {
		queue.Rerun = false
	}
	if queue.Slots == 0 {
		queue.Slots = 0
	}
	if queue.TmpDir == "" {
		queue.TmpDir = "/tmp"
	}
	if queue.Shell == "" {
		queue.Shell = "/bin/sh"
	}
	if queue.Prolog == "" {
		queue.Prolog = "NONE"
	}
	if queue.Epilog == "" {
		queue.Epilog = "NONE"
	}
	if queue.ShellStartMode == "" {
		queue.ShellStartMode = "unix_behavior"
	}
	if queue.StarterMethod == "" {
		queue.StarterMethod = "NONE"
	}
	if queue.SuspendMethod == "" {
		queue.SuspendMethod = "NONE"
	}
	if queue.ResumeMethod == "" {
		queue.ResumeMethod = "NONE"
	}
	if queue.TerminateMethod == "" {
		queue.TerminateMethod = "NONE"
	}
	if queue.Notify == "" {
		queue.Notify = "00:00:60"
	}
	if queue.OwnerList == "" {
		queue.OwnerList = "NONE"
	}
	if queue.UserLists == "" {
		queue.UserLists = "NONE"
	}
	if queue.XUserLists == "" {
		queue.XUserLists = "NONE"
	}
	if queue.SubordinateList == "" {
		queue.SubordinateList = "NONE"
	}
	if queue.ComplexValues == "" {
		queue.ComplexValues = "NONE"
	}
	if queue.Projects == "" {
		queue.Projects = "NONE"
	}
	if queue.XProjects == "" {
		queue.XProjects = "NONE"
	}
	if queue.Calendar == "" {
		queue.Calendar = "NONE"
	}
	if queue.InitialState == "" {
		queue.InitialState = "default"
	}
	if queue.SRt == "" {
		queue.SRt = "INFINITY"
	}
	if queue.HRt == "" {
		queue.HRt = "INFINITY"
	}
	if queue.SCpu == "" {
		queue.SCpu = "INFINITY"
	}
	if queue.HCpu == "" {
		queue.HCpu = "INFINITY"
	}
	if queue.SSize == "" {
		queue.SSize = "INFINITY"
	}
	if queue.HSize == "" {
		queue.HSize = "INFINITY"
	}
	if queue.SData == "" {
		queue.SData = "INFINITY"
	}
	if queue.HData == "" {
		queue.HData = "INFINITY"
	}
	if queue.SStack == "" {
		queue.SStack = "INFINITY"
	}
	if queue.HStack == "" {
		queue.HStack = "INFINITY"
	}
	if queue.SCore == "" {
		queue.SCore = "INFINITY"
	}
	if queue.HCore == "" {
		queue.HCore = "INFINITY"
	}
	if queue.SRss == "" {
		queue.SRss = "INFINITY"
	}
	if queue.HRss == "" {
		queue.HRss = "INFINITY"
	}
	if queue.SVmem == "" {
		queue.SVmem = "INFINITY"
	}
	if queue.HVmem == "" {
		queue.HVmem = "INFINITY"
	}
}

// AddClusterQueue adds a cluster queue.
func (c *CommandLineQConf) AddClusterQueue(queue ClusterQueueConfig) error {
	SetDefaultQueueValues(&queue)

	file, err := createTempDirWithFileName(queue.QName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("qname             %s\n", queue.QName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("hostlist          %s\n", queue.HostList))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("seq_no             %d\n", queue.SeqNo))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("load_thresholds    %s\n", queue.LoadThresholds))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_thresholds %s\n", queue.SuspendThresholds))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("nsuspend          %d\n", queue.NSuspend))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_interval   %s\n", queue.SuspendInterval))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("priority          %d\n", queue.Priority))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("min_cpu_interval    %s\n", queue.MinCpuInterval))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("processors        %s\n", queue.Processors))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("qtype             %s\n", queue.QType))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("ckpt_list          %s\n", queue.CkptList))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("pe_list            %s\n", queue.PeList))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("rerun             %v\n", queue.Rerun))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("slots             %d\n", queue.Slots))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("tmpdir            %s\n", queue.TmpDir))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("shell             %s\n", queue.Shell))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("prolog            %s\n", queue.Prolog))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("epilog            %s\n", queue.Epilog))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("shell_start_mode    %s\n", queue.ShellStartMode))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("starter_method     %s\n", queue.StarterMethod))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_method     %s\n", queue.SuspendMethod))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("resume_method      %s\n", queue.ResumeMethod))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("terminate_method   %s\n", queue.TerminateMethod))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("notify            %s\n", queue.Notify))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("owner_list         %s\n", queue.OwnerList))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("user_lists          %s\n", queue.UserLists))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xuser_lists         %s\n", queue.XUserLists))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("subordinate_list   %s\n", queue.SubordinateList))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("complex_values     %s\n", queue.ComplexValues))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("projects          %s\n", queue.Projects))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xprojects         %s\n", queue.XProjects))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("calendar          %s\n", queue.Calendar))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("initial_state      %s\n", queue.InitialState))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_rt               %s\n", queue.SRt))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_rt               %s\n", queue.HRt))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_cpu              %s\n", queue.SCpu))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_cpu              %s\n", queue.HCpu))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_fsize            %s\n", queue.SSize))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_fsize            %s\n", queue.HSize))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_data             %s\n", queue.SData))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_data             %s\n", queue.HData))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_stack            %s\n", queue.SStack))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_stack            %s\n", queue.HStack))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_core             %s\n", queue.SCore))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_core             %s\n", queue.HCore))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_rss              %s\n", queue.SRss))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_rss              %s\n", queue.HRss))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_vmem             %s\n", queue.SVmem))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_vmem             %s\n", queue.HVmem))
	if err != nil {
		return err
	}

	file.Close()

	_, err = c.RunCommand("-Aq", file.Name())
	return err
}

// DeleteClusterQueue deletes a cluster queue.
func (c *CommandLineQConf) DeleteClusterQueue(queueName string) error {
	_, err := c.RunCommand("-dq", queueName)
	return err
}

// ShowClusterQueue shows the specified cluster queue.
func (c *CommandLineQConf) ShowClusterQueue(queueName string) (ClusterQueueConfig, error) {
	out, err := c.RunCommand("-sq", queueName)
	if err != nil {
		return ClusterQueueConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := ClusterQueueConfig{QName: queueName}
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "qname":
			cfg.QName = fields[1]
		case "hostlist":
			cfg.HostList, _ = ParseMultiLineValue(lines, i)
		case "seq_no":
			cfg.SeqNo, _ = strconv.Atoi(fields[1])
		case "load_thresholds":
			cfg.LoadThresholds = fields[1]
		case "suspend_thresholds":
			cfg.SuspendThresholds = fields[1]
		case "nsuspend":
			cfg.NSuspend, _ = strconv.Atoi(fields[1])
		case "suspend_interval":
			cfg.SuspendInterval = fields[1]
		case "priority":
			cfg.Priority, _ = strconv.Atoi(fields[1])
		case "min_cpu_interval":
			cfg.MinCpuInterval = fields[1]
		case "processors":
			cfg.Processors = fields[1]
		case "qtype":
			cfg.QType = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		case "ckpt_list":
			cfg.CkptList, _ = ParseMultiLineValue(lines, i)
		case "pe_list":
			cfg.PeList, _ = ParseMultiLineValue(lines, i)
		case "rerun":
			cfg.Rerun, _ = strconv.ParseBool(fields[1])
		case "slots":
			cfg.Slots, _ = strconv.Atoi(fields[1])
		case "tmpdir":
			cfg.TmpDir = fields[1]
		case "shell":
			cfg.Shell = fields[1]
		case "prolog":
			cfg.Prolog = fields[1]
		case "epilog":
			cfg.Epilog = fields[1]
		case "shell_start_mode":
			cfg.ShellStartMode = fields[1]
		case "starter_method":
			cfg.StarterMethod = fields[1]
		case "suspend_method":
			cfg.SuspendMethod = fields[1]
		case "resume_method":
			cfg.ResumeMethod = fields[1]
		case "terminate_method":
			cfg.TerminateMethod = fields[1]
		case "notify":
			cfg.Notify = fields[1]
		case "owner_list":
			cfg.OwnerList, _ = ParseMultiLineValue(lines, i)
		case "user_lists":
			cfg.UserLists, _ = ParseMultiLineValue(lines, i)
		case "xuser_lists":
			cfg.XUserLists, _ = ParseMultiLineValue(lines, i)
		case "subordinate_list":
			cfg.SubordinateList, _ = ParseMultiLineValue(lines, i)
		case "complex_values":
			cfg.ComplexValues, _ = ParseMultiLineValue(lines, i)
		case "projects":
			cfg.Projects, _ = ParseMultiLineValue(lines, i)
		case "xprojects":
			cfg.XProjects, _ = ParseMultiLineValue(lines, i)
		case "calendar":
			cfg.Calendar, _ = ParseMultiLineValue(lines, i)
		case "initial_state":
			cfg.InitialState, _ = ParseMultiLineValue(lines, i)
		case "s_rt":
			cfg.SRt = fields[1]
		case "h_rt":
			cfg.HRt = fields[1]
		case "s_cpu":
			cfg.SCpu = fields[1]
		case "h_cpu":
			cfg.HCpu = fields[1]
		case "s_fsize":
			cfg.SSize = fields[1]
		case "h_fsize":
			cfg.HSize = fields[1]
		case "s_data":
			cfg.SData = fields[1]
		case "h_data":
			cfg.HData = fields[1]
		case "s_stack":
			cfg.SStack = fields[1]
		case "h_stack":
			cfg.HStack = fields[1]
		case "s_core":
			cfg.SCore = fields[1]
		case "h_core":
			cfg.HCore = fields[1]
		case "s_rss":
			cfg.SRss = fields[1]
		case "h_rss":
			cfg.HRss = fields[1]
		case "s_vmem":
			cfg.SVmem = fields[1]
		case "h_vmem":
			cfg.HVmem = fields[1]
		}
	}
	return cfg, nil
}

// ShowClusterQueues shows all cluster queues.
func (c *CommandLineQConf) ShowClusterQueues() ([]string, error) {
	output, err := c.RunCommand("-sql")
	if err != nil {
		return nil, err
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

// AddSubmitHosts adds a list of submit hosts.
func (c *CommandLineQConf) AddSubmitHosts(hostnames []string) error {
	_, err := c.RunCommand("-as", strings.Join(hostnames, ","))
	return err
}

// DeleteSubmitHost deletes a list of submit hosts.
func (c *CommandLineQConf) DeleteSubmitHost(hostnames []string) error {
	_, err := c.RunCommand("-ds", strings.Join(hostnames, ","))
	return err
}

// ShowSubmitHosts shows all submit hosts.
func (c *CommandLineQConf) ShowSubmitHosts() ([]string, error) {
	output, err := c.RunCommand("-ss")
	if err != nil {
		return nil, fmt.Errorf("error showing submit hosts: %v", err)
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

// AddUserToUserSetList adds a user to a user set list.
func (c *CommandLineQConf) AddUserToUserSetList(userList, listnameList string) error {
	_, err := c.RunCommand("-au", userList, listnameList)
	return err
}

// DeleteUserFromUserSetList deletes a user from a user set list.
func (c *CommandLineQConf) DeleteUserFromUserSetList(userList, listnameList string) error {
	_, err := c.RunCommand("-du", userList, listnameList)
	return err
}

// DeleteUserSetList deletes a user set list.
func (c *CommandLineQConf) DeleteUserSetList(userList string) error {
	_, err := c.RunCommand("-dul", userList)
	return err
}

// ShowUserSetList shows the specified user set list.
func (c *CommandLineQConf) ShowUserSetList(listnameList string) (UserSetListConfig, error) {
	out, err := c.RunCommand("-su", listnameList)
	if err != nil {
		return UserSetListConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := UserSetListConfig{Name: listnameList}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "name":
			cfg.Name = fields[1]
		case "type":
			cfg.Type = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		case "fshare":
			cfg.FShare, _ = strconv.Atoi(fields[1])
		case "oticket":
			cfg.OTicket, _ = strconv.Atoi(fields[1])
		case "entries":
			cfg.Entries = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		}
	}
	return cfg, nil
}

func SetDefaultUserSetListConfig(u *UserSetListConfig) {
	if u.Entries == "" {
		u.Entries = "NONE"
	}
	if u.Type == "" {
		u.Type = "ACL DEPT"
	}
}

func (c *CommandLineQConf) AddUserSetList(userSetListName string, u UserSetListConfig) error {
	SetDefaultUserSetListConfig(&u)

	file, err := createTempDirWithFileName(userSetListName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	if u.Entries == "" {
		u.Entries = "NONE"
	}

	_, err = file.WriteString(fmt.Sprintf("name           %s\n", u.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("type           %s\n", u.Type))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("fshare           %d\n", u.FShare))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("oticket           %d\n", u.OTicket))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("entries           %s\n", u.Entries))
	if err != nil {
		return err
	}
	file.Close()
	_, err = c.RunCommand("-Au", file.Name())
	return err
}

// AddUser adds a new user.
func (c *CommandLineQConf) AddUser(userConfig UserConfig) error {
	file, err := createTempDirWithFileName(userConfig.Name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("name           %s\n", userConfig.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("oticket        %d\n", userConfig.OTicket))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("fshare         %d\n", userConfig.FShare))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("delete_time     %d\n", userConfig.DeleteTime))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("default_project %s\n", userConfig.DefaultProject))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Auser", file.Name())
	return err
}

// DeleteUser deletes a list of users.
func (c *CommandLineQConf) DeleteUser(users []string) error {
	_, err := c.RunCommand("-duser", strings.Join(users, ","))
	return err
}

// ShowUser shows the specified user.
func (c *CommandLineQConf) ShowUser(userName string) (UserConfig, error) {
	out, err := c.RunCommand("-suser", userName)
	if err != nil {
		return UserConfig{}, err
	}
	// CS-464 - Inconsistent exist code. Is 0 but should be 1
	// if a user is not defined.
	if strings.Contains(out, "is not known as user") {
		return UserConfig{}, fmt.Errorf("user %s is not defined", userName)
	}

	lines := strings.Split(out, "\n")
	cfg := UserConfig{Name: userName}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "name":
			cfg.Name = fields[1]
		case "oticket":
			cfg.OTicket, _ = strconv.Atoi(fields[1])
		case "fshare":
			cfg.FShare, _ = strconv.Atoi(fields[1])
		case "delete_time":
			cfg.DeleteTime, _ = strconv.Atoi(fields[1])
		case "default_project":
			cfg.DefaultProject = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		}
	}
	return cfg, nil
}

func (c *CommandLineQConf) ShowUsers() ([]string, error) {
	output, err := c.RunCommand("-suserl")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

// ShowUsers shows all user set lists.
func (c *CommandLineQConf) ShowUserSetLists() ([]string, error) {
	output, err := c.RunCommand("-sul")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

// ClearUsage clears all user/project sharetree usage.
func (c *CommandLineQConf) ClearUsage() error {
	_, err := c.RunCommand("-clearusage")
	return err
}

// CleanQueue cleans a queue for the specified destinations.
func (c *CommandLineQConf) CleanQueue(destinID []string) error {
	_, err := c.RunCommand("-cq", strings.Join(destinID, ","))
	return err
}

// ShutdownExecDaemons shuts down execution daemons for the specified hosts.
func (c *CommandLineQConf) ShutdownExecDaemons(hosts []string) error {
	_, err := c.RunCommand("-kej", strings.Join(hosts, ","))
	return err
}

// ShutdownMasterDaemon shuts down the master daemon.
func (c *CommandLineQConf) ShutdownMasterDaemon() error {
	_, err := c.RunCommand("-km")
	return err
}

// ShutdownSchedulingDaemon shuts down the scheduling daemon.
func (c *CommandLineQConf) ShutdownSchedulingDaemon() error {
	_, err := c.RunCommand("-ks")
	return err
}

// KillEventClient kills the event clients with the specified event IDs.
func (c *CommandLineQConf) KillEventClient(evids []string) error {
	_, err := c.RunCommand("-kec", strings.Join(evids, ","))
	return err
}

// KillQmasterThread kills the specified qmaster thread.
func (c *CommandLineQConf) KillQmasterThread(threadName string) error {
	_, err := c.RunCommand("-kt", threadName)
	return err
}

// ModifyAttribute modifies an attribute of an object.
func (c *CommandLineQConf) ModifyAttribute(objName, attrName, val, objIDList string) error {
	_, err := c.RunCommand("-mattr", objName, attrName, val, objIDList)
	return err
}

func (c *CommandLineQConf) AddAttribute(objName, attrName, val, objIDList string) error {
	_, err := c.RunCommand("-aattr", objName, attrName, val, objIDList)
	return err
}

// ModifyAllComplexes modifies complex attributes.
func (c *CommandLineQConf) ModifyAllComplexes(centries []ComplexEntryConfig) error {
	// Create a temporary file with the complex attributes configuration
	file, err := os.CreateTemp("", "complexes")
	if err != nil {
		return err
	}
	defer file.Name()

	for _, resource := range centries {
		_, err = file.WriteString(fmt.Sprintf("%s %s %s %s %s %s %s %d\n",
			resource.Name, resource.Shortcut, resource.Type, resource.Relop, resource.Requestable,
			resource.Consumable, resource.Default, resource.Urgency))
		if err != nil {
			return err
		}
	}
	file.Close()

	_, err = c.RunCommand("-Mc", file.Name())
	return err
}

// ModifyCalendar modifies a calendar.
func (c *CommandLineQConf) ModifyCalendar(calendarName string, cfg CalendarConfig) error {
	if cfg.Year == "" {
		cfg.Year = "NONE"
	}
	if cfg.Week == "" {
		cfg.Week = "NONE"
	}

	file, err := createTempDirWithFileName(calendarName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("calendar_name    %s\n", calendarName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("year             %s\n", cfg.Year))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("week             %s\n", cfg.Week))
	if err != nil {
		return err
	}
	file.Close()
	_, err = c.RunCommand("-Mcal", file.Name())
	return err
}

// ModifyCkptInterface modifies a checkpointing interface.
func (c *CommandLineQConf) ModifyCkptInterface(ckptName string, cfg CkptInterfaceConfig) error {
	file, err := createTempDirWithFileName(ckptName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("ckpt_name           %s\n", ckptName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("interface      %s\n", cfg.Interface))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("clean_command   %s\n", cfg.CleanCommand))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("ckpt_command  %s\n", cfg.CheckpointCmd))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("migr_command    %s\n", cfg.MigrCommand))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("restart_command %s\n", cfg.RestartCommand))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("ckpt_dir        %s\n", cfg.CkptDir))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("signal   %s\n", cfg.Signal))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("when   %s\n", cfg.When))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Mckpt", file.Name())
	return err
}

// ModifyHostConfiguration modifies a host configuration.
func (c *CommandLineQConf) ModifyHostConfiguration(configName string, cfg HostConfiguration) error {
	file, err := createTempDirWithFileName(configName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("mailer %s\n", cfg.Mailer))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xterm %s\n", cfg.Xterm))
	if err != nil {
		return err
	}
	file.Close()

	out, err := c.RunCommand("-Mconf", file.Name())
	if err != nil {
		return fmt.Errorf("error modifying host configuration (%s): %s",
			out, err)
	}
	return nil
}

func createTempDirWithFileName(name string) (*os.File, error) {
	dir, err := os.MkdirTemp("", name)
	if err != nil {
		return nil, err
	}

	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}
	return file, nil
}

// ModifyGlobalConfig modifies the global configuration.
func (c *CommandLineQConf) ModifyGlobalConfig(cfg GlobalConfig) error {
	file, err := createTempDirWithFileName("global")
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	v := reflect.ValueOf(cfg)
	typeOfS := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldName := typeOfS.Field(i).Tag.Get("json")
		fieldValue := v.Field(i).Interface()
		_, err = file.WriteString(fmt.Sprintf("%s %v\n", fieldName, fieldValue))
		if err != nil {
			return err
		}
	}
	file.Close()

	_, err = c.RunCommand("-Mconf", file.Name())
	return err
}

func SetDefaultExecHostConfig(cfg *HostExecConfig) {
	if cfg.LoadScaling == "" {
		cfg.LoadScaling = "NONE"
	}
	if cfg.ComplexValues == "" {
		cfg.ComplexValues = "NONE"
	}
	if cfg.UserLists == "" {
		cfg.UserLists = "NONE"
	}
	if cfg.XUserLists == "" {
		cfg.XUserLists = "NONE"
	}
	if cfg.Projects == "" {
		cfg.Projects = "NONE"
	}
	if cfg.XProjects == "" {
		cfg.XProjects = "NONE"
	}
	if cfg.UsageScaling == "" {
		cfg.UsageScaling = "NONE"
	}
	if cfg.ReportVariables == "" {
		cfg.ReportVariables = "NONE"
	}
}

// ModifyExecHost modifies an execution host.
func (c *CommandLineQConf) ModifyExecHost(execHostName string, cfg HostExecConfig) error {
	SetDefaultExecHostConfig(&cfg)
	file, err := createTempDirWithFileName(execHostName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("hostname         %s\n", execHostName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("load_scaling     %s\n", cfg.LoadScaling))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("complex_values   %s\n", cfg.ComplexValues))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("user_lists       %s\n", cfg.UserLists))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xuser_lists      %s\n", cfg.XUserLists))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("projects         %s\n", cfg.Projects))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xprojects        %s\n", cfg.XProjects))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("usage_scaling    %s\n", cfg.UsageScaling))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("report_variables %s\n", cfg.ReportVariables))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Me", file.Name())
	return err
}

// ModifyHostGroup modifies a host group.
func (c *CommandLineQConf) ModifyHostGroup(hostGroupName string, cfg HostGroupConfig) error {
	file, err := createTempDirWithFileName(hostGroupName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	if cfg.Hostlist == "" {
		cfg.Hostlist = "NONE"
	}

	_, err = file.WriteString(fmt.Sprintf("group_name %s\n", hostGroupName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("hostlist %s\n", cfg.Hostlist))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Mhgrp", file.Name())
	return err
}

func SetResourceQuotaSetDefaults(cfg *ResourceQuotaSetConfig) {
	if cfg.Description == "" {
		cfg.Description = "NONE"
	}
}

// ModifyResourceQuotaSet modifies a resource quota set.
func (c *CommandLineQConf) ModifyResourceQuotaSet(rqsName string, cfg ResourceQuotaSetConfig) error {
	SetResourceQuotaSetDefaults(&cfg)
	file, err := createTempDirWithFileName(rqsName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString("{\n")
	if err != nil {
		return err
	}

	_, err = file.WriteString(fmt.Sprintf("name         %s\n", rqsName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("description  %s\n", cfg.Description))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("enabled      %v\n", cfg.Enabled))
	if err != nil {
		return err
	}
	for _, limit := range cfg.Limits {
		_, err = file.WriteString(fmt.Sprintf("limit        %s\n", limit))
		if err != nil {
			return err
		}
	}
	_, err = file.WriteString("}\n")
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Mrqs", file.Name())
	return err
}

// ModifyParallelEnvironment modifies a parallel environment.
func (c *CommandLineQConf) ModifyParallelEnvironment(peName string, cfg ParallelEnvironmentConfig) error {
	SetDefaultParallelEnvironmentValues(&cfg)
	file, err := createTempDirWithFileName(peName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	err = writePE(file, cfg)
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Mp", file.Name())
	return err
}

// ModifyProject modifies a project.
func (c *CommandLineQConf) ModifyProject(projectName string, cfg ProjectConfig) error {
	SetDefaultProjectValues(&cfg)
	file, err := createTempDirWithFileName(projectName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("name    %s\n", projectName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("oticket %d\n", cfg.OTicket))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("fshare  %d\n", cfg.FShare))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("acl     %s\n", cfg.ACL))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xacl    %s\n", cfg.XACL))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Mprj", file.Name())
	return err
}

// ModifyClusterQueue modifies a cluster queue.
func (c *CommandLineQConf) ModifyClusterQueue(queueName string, cfg ClusterQueueConfig) error {
	SetDefaultQueueValues(&cfg)
	file, err := createTempDirWithFileName(queueName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("qname             %s\n", queueName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("hostlist          %s\n", cfg.HostList))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("seq_no            %d\n", cfg.SeqNo))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("load_thresholds    %s\n", cfg.LoadThresholds))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_thresholds %s\n", cfg.SuspendThresholds))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("nsuspend          %d\n", cfg.NSuspend))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_interval   %s\n", cfg.SuspendInterval))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("priority          %d\n", cfg.Priority))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("min_cpu_interval    %s\n", cfg.MinCpuInterval))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("processors        %s\n", cfg.Processors))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("qtype             %s\n", cfg.QType))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("ckpt_list          %s\n", cfg.CkptList))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("pe_list            %s\n", cfg.PeList))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("rerun             %v\n", cfg.Rerun))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("slots             %d\n", cfg.Slots))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("tmpdir            %s\n", cfg.TmpDir))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("shell             %s\n", cfg.Shell))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("prolog            %s\n", cfg.Prolog))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("epilog            %s\n", cfg.Epilog))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("shell_start_mode    %s\n", cfg.ShellStartMode))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("starter_method     %s\n", cfg.StarterMethod))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_method     %s\n", cfg.SuspendMethod))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("resume_method      %s\n", cfg.ResumeMethod))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("terminate_method   %s\n", cfg.TerminateMethod))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("notify            %s\n", cfg.Notify))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("owner_list         %s\n", cfg.OwnerList))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("user_lists          %s\n", cfg.UserLists))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xuser_lists         %s\n", cfg.XUserLists))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("subordinate_list   %s\n", cfg.SubordinateList))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("complex_values     %s\n", cfg.ComplexValues))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("projects          %s\n", cfg.Projects))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xprojects         %s\n", cfg.XProjects))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("calendar          %s\n", cfg.Calendar))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("initial_state      %s\n", cfg.InitialState))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_rt               %s\n", cfg.SRt))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_rt               %s\n", cfg.HRt))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_cpu              %s\n", cfg.SCpu))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_cpu              %s\n", cfg.HCpu))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_fsize            %s\n", cfg.SSize))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_fsize            %s\n", cfg.HSize))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_data             %s\n", cfg.SData))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_data             %s\n", cfg.HData))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_stack            %s\n", cfg.SStack))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_stack            %s\n", cfg.HStack))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_core             %s\n", cfg.SCore))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_core             %s\n", cfg.HCore))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_rss              %s\n", cfg.SRss))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_rss              %s\n", cfg.HRss))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_vmem             %s\n", cfg.SVmem))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_vmem             %s\n", cfg.HVmem))
	if err != nil {
		return err
	}

	file.Close()

	_, err = c.RunCommand("-Mq", file.Name())
	return err
}

// ModifyUserset modifies a user set list.
func (c *CommandLineQConf) ModifyUserset(listnameList string, cfg UserSetListConfig) error {
	file, err := createTempDirWithFileName(listnameList)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	if cfg.Entries == "" {
		cfg.Entries = "NONE"
	}

	_, err = file.WriteString(fmt.Sprintf("name    %s\n", listnameList))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("type    %s\n", cfg.Type))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("fshare  %d\n", cfg.FShare))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("oticket %d\n", cfg.OTicket))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("entries %s\n", cfg.Entries))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Mu", file.Name())
	return err
}

func SetDefaultUserValues(u *UserConfig) {
	if u.DefaultProject == "" {
		u.DefaultProject = "NONE"
	}
}

// ModifyUser modifies a user.
func (c *CommandLineQConf) ModifyUser(userName string, cfg UserConfig) error {
	SetDefaultUserValues(&cfg)
	file, err := createTempDirWithFileName(userName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("name           %s\n", userName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("oticket        %d\n", cfg.OTicket))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("fshare         %d\n", cfg.FShare))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("delete_time     %d\n", cfg.DeleteTime))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("default_project %s\n", cfg.DefaultProject))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Muser", file.Name())
	return err
}

// DeleteAttribute deletes an attribute from an object.
func (c *CommandLineQConf) DeleteAttribute(objName, attrName, val, objIDList string) error {
	_, err := c.RunCommand("-dattr", objName, attrName, val, objIDList)
	return err
}
