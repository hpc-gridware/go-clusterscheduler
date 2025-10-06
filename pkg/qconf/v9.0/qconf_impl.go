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
	"time"
)

type CommandLineQConf struct {
	config CommandLineQConfConfig
}

type CommandLineQConfConfig struct {
	Executable string
	DryRun     bool
	// DelayAfter is the time to wait after executing a command.
	// This is useful for not overloading qmaster when 1000s of
	// configuration objects (like queues) are defined.
	DelayAfter time.Duration
}

// NewCommandLineQConf creates a new instance of CommandLineQConf.
func NewCommandLineQConf(config CommandLineQConfConfig) (*CommandLineQConf, error) {
	if config.Executable == "" {
		config.Executable = "qconf"
	}
	return &CommandLineQConf{config: config}, nil
}

// RunCommand executes the qconf command with the specified arguments.
func (c *CommandLineQConf) RunCommand(args ...string) (string, error) {
	if c.config.DryRun {
		fmt.Printf("Executing: %s, %v", c.config.Executable, args)
		return "", nil
	}
	cmd := exec.Command(c.config.Executable, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	// Set the SGE_SINGLE_LINE environment variable to true to ensure that
	// qconf returns a single line of output for each entry.
	cmd.Env = append(cmd.Environ(), "SGE_SINGLE_LINE=true")
	err := cmd.Run()
	if c.config.DelayAfter != 0 {
		<-time.After(c.config.DelayAfter)
	}
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

func GetEnvironment() (*ClusterEnvironment, error) {
	clusterEnvironment := ClusterEnvironment{}
	clusterEnvironment.Name = os.Getenv("SGE_CLUSTER_NAME")
	clusterEnvironment.Root = os.Getenv("SGE_ROOT")
	clusterEnvironment.Cell = os.Getenv("SGE_CELL")
	clusterEnvironment.QmasterPort, _ = GetEnvInt("SGE_QMASTER_PORT")
	clusterEnvironment.ExecdPort, _ = GetEnvInt("SGE_EXECD_PORT")
	return &clusterEnvironment, nil
}

func (c *CommandLineQConf) GetClusterConfiguration() (ClusterConfig, error) {
	cc := ClusterConfig{}

	// general settings which defines the environment
	var err error
	cc.ClusterEnvironment, err = GetEnvironment()
	if err != nil {
		return cc, fmt.Errorf("failed to read cluster environment: %v", err)
	}

	cc.GlobalConfig, err = c.ShowGlobalConfiguration()
	if err != nil {
		return cc, fmt.Errorf("failed to read global config: %v", err)
	}

	cc.SchedulerConfig, err = c.ShowSchedulerConfiguration()
	if err != nil {
		return cc, fmt.Errorf("failed to read scheduler config: %v", err)
	}

	hostConfigs, err := c.ShowHostConfigurations()
	if err != nil {
		return cc, fmt.Errorf("failed to read host configs: %v", err)
	}
	cc.HostConfigurations = make(map[string]HostConfiguration)
	for _, host := range hostConfigs {
		hc, err := c.ShowHostConfiguration(host)
		if err != nil {
			// host is might be unreachable
			fmt.Printf("warning: host %s is unreachable\n", host)
			continue
		}
		cc.HostConfigurations[host] = hc
	}

	projectNames, err := c.ShowProjects()
	if err != nil {
		return cc, fmt.Errorf("failed to read projects: %v", err)
	}
	cc.Projects = make(map[string]ProjectConfig)
	for _, projectName := range projectNames {
		pc, err := c.ShowProject(projectName)
		if err != nil {
			return cc, fmt.Errorf("failed to read project: %v", err)
		}
		cc.Projects[projectName] = pc
	}

	// Read Calendars
	calendars, err := c.ShowCalendars()
	if err != nil {
		return cc, fmt.Errorf("failed to read calendars: %v", err)
	}
	cc.Calendars = make(map[string]CalendarConfig)
	for _, calendar := range calendars {
		ccal, err := c.ShowCalendar(calendar)
		if err != nil {
			return cc, fmt.Errorf("failed to read calendar: %v", err)
		}
		cc.Calendars[calendar] = ccal
	}

	// Read Complex Entries
	complexes, err := c.ShowAllComplexes()
	if err != nil {
		return cc, fmt.Errorf("failed to read complex entries: %v", err)
	}
	cc.ComplexEntries = make(map[string]ComplexEntryConfig)
	for _, complex := range complexes {
		cc.ComplexEntries[complex.Name] = complex
	}

	// Read Ckpt Interfaces
	ckptInterfaces, err := c.ShowCkptInterfaces()
	if err != nil {
		return cc, fmt.Errorf("failed to read ckpt interfaces: %v", err)
	}
	cc.CkptInterfaces = make(map[string]CkptInterfaceConfig, 0)
	for _, ckptInterface := range ckptInterfaces {
		ci, err := c.ShowCkptInterface(ckptInterface)
		if err != nil {
			return cc, fmt.Errorf("failed to read ckpt interface: %v", err)
		}
		cc.CkptInterfaces[ckptInterface] = ci
	}

	// Read Exec Hosts
	execHosts, err := c.ShowExecHosts()
	if err != nil {
		return cc, fmt.Errorf("failed to read exec hosts: %v", err)
	}
	cc.ExecHosts = make(map[string]HostExecConfig, 0)
	for _, execHost := range execHosts {
		eh, err := c.ShowExecHost(execHost)
		if err != nil {
			fmt.Printf("warning: exec host %s is unreachable\n", execHost)
			continue
		}
		cc.ExecHosts[execHost] = eh
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
	cc.HostGroups = make(map[string]HostGroupConfig, 0)
	for _, hostGroup := range hostGroups {
		hg, err := c.ShowHostGroup(hostGroup)
		if err != nil {
			return cc, fmt.Errorf("failed to read host group: %v", err)
		}
		cc.HostGroups[hostGroup] = hg
	}

	// Read Resource Quota Sets
	resourceQuotaSets, err := c.ShowResourceQuotaSets()
	if err != nil {
		return cc, fmt.Errorf("failed to read resource quota sets: %v", err)
	}
	cc.ResourceQuotaSets = make(map[string]ResourceQuotaSetConfig, 0)
	for _, resourceQuotaSet := range resourceQuotaSets {
		rqs, err := c.ShowResourceQuotaSet(resourceQuotaSet)
		if err != nil {
			return cc, fmt.Errorf("failed to read resource quota set: %v", err)
		}
		cc.ResourceQuotaSets[resourceQuotaSet] = rqs
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
	cc.ParallelEnvironments = make(map[string]ParallelEnvironmentConfig, 0)
	for _, parallelEnvironment := range parallelEnvironments {
		pe, err := c.ShowParallelEnvironment(parallelEnvironment)
		if err != nil {
			return cc, fmt.Errorf("failed to read parallel environment: %v", err)
		}
		cc.ParallelEnvironments[parallelEnvironment] = pe
	}

	// Read Users
	users, err := c.ShowUsers()
	if err != nil {
		return cc, fmt.Errorf("failed to read users: %v", err)
	}
	cc.Users = make(map[string]UserConfig, 0)
	for _, user := range users {
		u, err := c.ShowUser(user)
		if err != nil {
			return cc, fmt.Errorf("failed to read user: %v", err)
		}
		cc.Users[user] = u
	}

	// Read Cluster Queues
	clusterQueues, err := c.ShowClusterQueues()
	if err != nil {
		return cc, fmt.Errorf("failed to read cluster queues: %v", err)
	}
	cc.ClusterQueues = make(map[string]ClusterQueueConfig, 0)
	for _, clusterQueue := range clusterQueues {
		cq, err := c.ShowClusterQueue(clusterQueue)
		if err != nil {
			return cc, fmt.Errorf("failed to read cluster queue: %v", err)
		}
		cc.ClusterQueues[clusterQueue] = cq
	}

	// Read User Set Lists
	userSetLists, err := c.ShowUserSetLists()
	if err != nil {
		return cc, fmt.Errorf("failed to read user set lists: %v", err)
	}
	cc.UserSetLists = make(map[string]UserSetListConfig, 0)
	for _, userSetList := range userSetLists {
		usl, err := c.ShowUserSetList(userSetList)
		if err != nil {
			return cc, fmt.Errorf("failed to read user set list: %v", err)
		}
		cc.UserSetLists[userSetList] = usl
	}

	return cc, nil
}

func (c *CommandLineQConf) ApplyClusterConfiguration(cc ClusterConfig) error {
	// make plan
	// apply plan
	return nil
}

// AddCalendar adds a new calendar.
func (c *CommandLineQConf) AddCalendar(cfg CalendarConfig) error {
	// Create file in tmp directory with calendar configuration.
	// Use the file as input to the qconf command.
	// Remove the file after the command completes.

	if cfg.Name == "" {
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

	_, err = file.WriteString(fmt.Sprintf("calendar_name    %s\n", cfg.Name))
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
	cfg := CalendarConfig{Name: calendarName}
	for _, line := range lines {
		if strings.HasPrefix(line, "year") {
			cfg.Year = strings.TrimSpace(strings.Fields(line)[1])
		} else if strings.HasPrefix(line, "week") {
			cfg.Week = strings.TrimSpace(strings.Fields(line)[1])
		} else if strings.HasPrefix(line, "calendar_name") {
			cfg.Name = strings.TrimSpace(strings.Fields(line)[1])
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
		c.Requestable = ConsumableNO
	}
	if c.Consumable == "" {
		c.Consumable = ConsumableNO
	}
	if c.Default == "" {
		if c.Type == ResourceTypeString {
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
	fmt.Printf("file: %s\n", file.Name())
	//defer os.RemoveAll(filepath.Dir(file.Name()))

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
	_, err = file.WriteString(fmt.Sprintf("ckpt_command  %s\n", cfg.CheckpointCommand))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("ckpt_dir      %s\n", cfg.CheckpointDir))
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
			cfg.CheckpointCommand = fields[1]
		case "migr_command":
			cfg.MigrCommand = fields[1]
		case "restart_command":
			cfg.RestartCommand = fields[1]
		case "ckpt_dir":
			cfg.CheckpointDir = fields[1]
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
		return fmt.Errorf("hostname not set in host configuration")
	}

	file, err := createTempDirWithFileName(config.Name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	// Handle pointer fields properly - only write if not nil
	if config.ExecdSpoolDir != nil {
		if _, err = file.WriteString(fmt.Sprintf("execd_spool_dir %s\n", *config.ExecdSpoolDir)); err != nil {
			return err
		}
	}
	if config.Mailer != nil {
		if _, err = file.WriteString(fmt.Sprintf("mailer %s\n", *config.Mailer)); err != nil {
			return err
		}
	}
	if config.Xterm != nil {
		if _, err = file.WriteString(fmt.Sprintf("xterm %s\n", *config.Xterm)); err != nil {
			return err
		}
	}

	// Handle slice fields
	for _, sensor := range config.LoadSensors {
		if _, err = file.WriteString(fmt.Sprintf("load_sensor %s\n", sensor)); err != nil {
			return err
		}
	}

	if config.Prolog != nil {
		if _, err = file.WriteString(fmt.Sprintf("prolog %s\n", *config.Prolog)); err != nil {
			return err
		}
	}
	if config.Epilog != nil {
		if _, err = file.WriteString(fmt.Sprintf("epilog %s\n", *config.Epilog)); err != nil {
			return err
		}
	}
	if config.ShellStartMode != nil {
		if _, err = file.WriteString(fmt.Sprintf("shell_start_mode %s\n", *config.ShellStartMode)); err != nil {
			return err
		}
	}

	// Write login shells
	for _, shell := range config.LoginShells {
		if _, err = file.WriteString(fmt.Sprintf("login_shells %s\n", shell)); err != nil {
			return err
		}
	}

	if config.LoadReportTime != nil {
		if _, err = file.WriteString(fmt.Sprintf("load_report_time %s\n", *config.LoadReportTime)); err != nil {
			return err
		}
	}
	if config.SetTokenCmd != nil {
		if _, err = file.WriteString(fmt.Sprintf("set_token_cmd %s\n", *config.SetTokenCmd)); err != nil {
			return err
		}
	}
	if config.PagCmd != nil {
		if _, err = file.WriteString(fmt.Sprintf("pag_cmd %s\n", *config.PagCmd)); err != nil {
			return err
		}
	}
	if config.TokenExtendTime != nil {
		if _, err = file.WriteString(fmt.Sprintf("token_extend_time %s\n", *config.TokenExtendTime)); err != nil {
			return err
		}
	}
	if config.ShepherdCmd != nil {
		if _, err = file.WriteString(fmt.Sprintf("shepherd_cmd %s\n", *config.ShepherdCmd)); err != nil {
			return err
		}
	}

	// Write execd params
	for _, param := range config.ExecdParams {
		if _, err = file.WriteString(fmt.Sprintf("execd_params %s\n", param)); err != nil {
			return err
		}
	}

	// Write reporting params
	for _, param := range config.ReportingParams {
		if _, err = file.WriteString(fmt.Sprintf("reporting_params %s\n", param)); err != nil {
			return err
		}
	}

	// Write gid range
	for _, gid := range config.GidRange {
		if _, err = file.WriteString(fmt.Sprintf("gid_range %s\n", gid)); err != nil {
			return err
		}
	}

	if config.QloginDaemon != nil {
		if _, err = file.WriteString(fmt.Sprintf("qlogin_daemon %s\n", *config.QloginDaemon)); err != nil {
			return err
		}
	}
	if config.QloginCommand != nil {
		if _, err = file.WriteString(fmt.Sprintf("qlogin_command %s\n", *config.QloginCommand)); err != nil {
			return err
		}
	}
	if config.RshDaemon != nil {
		if _, err = file.WriteString(fmt.Sprintf("rsh_daemon %s\n", *config.RshDaemon)); err != nil {
			return err
		}
	}
	if config.RshCommand != nil {
		if _, err = file.WriteString(fmt.Sprintf("rsh_command %s\n", *config.RshCommand)); err != nil {
			return err
		}
	}
	if config.RloginDaemon != nil {
		if _, err = file.WriteString(fmt.Sprintf("rlogin_daemon %s\n", *config.RloginDaemon)); err != nil {
			return err
		}
	}
	if config.RloginCommand != nil {
		if _, err = file.WriteString(fmt.Sprintf("rlogin_command %s\n", *config.RloginCommand)); err != nil {
			return err
		}
	}
	if config.RescheduleUnknown != nil {
		if _, err = file.WriteString(fmt.Sprintf("reschedule_unknown %s\n", *config.RescheduleUnknown)); err != nil {
			return err
		}
	}
	if config.LibJvmPath != nil {
		if _, err = file.WriteString(fmt.Sprintf("libjvm_path %s\n", *config.LibJvmPath)); err != nil {
			return err
		}
	}
	if config.AdditionalJvmArgs != nil {
		if _, err = file.WriteString(fmt.Sprintf("additional_jvm_args %s\n", *config.AdditionalJvmArgs)); err != nil {
			return err
		}
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
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := fields[0]
		value := strings.Join(fields[1:], " ")

		switch key {
		case "execd_spool_dir":
			cfg.ExecdSpoolDir = &value
		case "mailer":
			cfg.Mailer = &value
		case "xterm":
			cfg.Xterm = &value
		case "load_sensor":
			cfg.LoadSensors = append(cfg.LoadSensors, value)
		case "prolog":
			cfg.Prolog = &value
		case "epilog":
			cfg.Epilog = &value
		case "shell_start_mode":
			cfg.ShellStartMode = &value
		case "login_shells":
			cfg.LoginShells = append(cfg.LoginShells, value)
		case "load_report_time":
			cfg.LoadReportTime = &value
		case "set_token_cmd":
			cfg.SetTokenCmd = &value
		case "pag_cmd":
			cfg.PagCmd = &value
		case "token_extend_time":
			cfg.TokenExtendTime = &value
		case "shepherd_cmd":
			cfg.ShepherdCmd = &value
		case "execd_params":
			cfg.ExecdParams = append(cfg.ExecdParams, value)
		case "reporting_params":
			cfg.ReportingParams = append(cfg.ReportingParams, value)
		case "gid_range":
			cfg.GidRange = append(cfg.GidRange, value)
		case "qlogin_daemon":
			cfg.QloginDaemon = &value
		case "qlogin_command":
			cfg.QloginCommand = &value
		case "rsh_daemon":
			cfg.RshDaemon = &value
		case "rsh_command":
			cfg.RshCommand = &value
		case "rlogin_daemon":
			cfg.RloginDaemon = &value
		case "rlogin_command":
			cfg.RloginCommand = &value
		case "reschedule_unknown":
			cfg.RescheduleUnknown = &value
		case "libjvm_path":
			cfg.LibJvmPath = &value
		case "additional_jvm_args":
			cfg.AdditionalJvmArgs = &value
		}
	}
	return cfg, nil
}

// ShowGlobalConfiguration shows the global configuration.
func (c *CommandLineQConf) ShowGlobalConfiguration() (*GlobalConfig, error) {
	out, err := c.RunCommand("-sconf", "global")
	if err != nil {
		return nil, err
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
			cfg.LoadSensors = ParseCommaSeparatedMultiLineValues(lines, i)
		case "prolog":
			cfg.Prolog = fields[1]
		case "epilog":
			cfg.Epilog = fields[1]
		case "shell_start_mode":
			cfg.ShellStartMode = fields[1]
		case "login_shells":
			cfg.LoginShells = ParseCommaSeparatedMultiLineValues(lines, i)
		case "min_uid":
			// Assuming the value can be converted to an integer
			cfg.MinUID, _ = strconv.Atoi(fields[1])
		case "min_gid":
			cfg.MinGID, _ = strconv.Atoi(fields[1])
		case "user_lists":
			cfg.UserLists = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "xuser_lists":
			cfg.XUserLists = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "projects":
			cfg.Projects = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "xprojects":
			cfg.XProjects = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "enforce_project":
			cfg.EnforceProject = fields[1]
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
			cfg.QmasterParams = ParseSpaceAndCommaSeparatedMultiLineValues(lines, i)
		case "execd_params":
			cfg.ExecdParams = ParseSpaceAndCommaSeparatedMultiLineValues(lines, i)
		case "reporting_params":
			cfg.ReportingParams = ParseSpaceAndCommaSeparatedMultiLineValues(lines, i)
		case "finished_jobs":
			cfg.FinishedJobs, _ = strconv.Atoi(fields[1])
		case "gid_range":
			cfg.GidRange = ParseCommaSeparatedMultiLineValues(lines, i)
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
			cfg.JsvAllowedMod = ParseCommaSeparatedMultiLineValues(lines, i)
		}
	}
	return &cfg, nil
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
	file, err := createTempDirWithFileName(hostExecConfig.Name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("hostname         %s\n",
		hostExecConfig.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("load_scaling     %s\n",
		JoinStringFloatMap(hostExecConfig.LoadScaling, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("complex_values   %s\n",
		JoinStringStringMap(hostExecConfig.ComplexValues, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("user_lists       %s\n",
		JoinList(hostExecConfig.UserLists, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xuser_lists      %s\n",
		JoinList(hostExecConfig.XUserLists, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("projects         %s\n",
		JoinList(hostExecConfig.Projects, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xprojects        %s\n",
		JoinList(hostExecConfig.XProjects, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("usage_scaling    %s\n",
		JoinStringFloatMap(hostExecConfig.UsageScaling, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("report_variables %s\n",
		JoinList(hostExecConfig.ReportVariables, ",")))
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

func ParseIntoStringFloatMap(val string, sep string) map[string]float64 {
	pairs := strings.Split(val, sep)
	out := make(map[string]float64)
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			continue
		}
		f, err := strconv.ParseFloat(kv[1], 64)
		if err != nil {
			continue
		}
		out[kv[0]] = f
	}
	return out
}

func ParseIntoStringStringMap(val string, sep string) map[string]string {
	pairs := strings.Split(val, sep)
	out := make(map[string]string)
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			continue
		}
		out[kv[0]] = kv[1]
	}
	return out
}

// ShowExecHost shows the specified execution host.
func (c *CommandLineQConf) ShowExecHost(hostName string) (HostExecConfig, error) {
	out, err := c.RunCommand("-se", hostName)
	if err != nil {
		return HostExecConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := HostExecConfig{Name: hostName}
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "hostname":
			cfg.Name = fields[1]
		case "load_scaling":
			line, _ := ParseMultiLineValue(lines, i)
			cfg.LoadScaling = ParseIntoStringFloatMap(line, ",")
		case "complex_values":
			line, _ := ParseMultiLineValue(lines, i)
			cfg.ComplexValues = ParseIntoStringStringMap(line, ",")
		case "user_lists":
			// https://linux.die.net/man/5/sge_host_conf
			cfg.UserLists = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "xuser_lists":
			cfg.XUserLists = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "projects":
			cfg.Projects = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "xprojects":
			cfg.XProjects = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "usage_scaling":
			line, _ := ParseMultiLineValue(lines, i)
			cfg.UsageScaling = ParseIntoStringFloatMap(line, ",")
		case "report_variables":
			cfg.ReportVariables = ParseCommaSeparatedMultiLineValues(lines, i)
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
	if hosts == nil {
		return nil
	}
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
	file, err := createTempDirWithFileName(hostGroup.Name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	if !strings.HasPrefix(hostGroup.Name, "@") {
		return fmt.Errorf("group name must start with '@'")
	}

	_, err = file.WriteString(fmt.Sprintf("group_name %s\n", hostGroup.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("hostlist %s\n",
		JoinList(hostGroup.Hosts, " ")))
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

// ShowHostGroup shows the host list of a particular host group. The host
// list can contain other host groups. Use ShowHowGroupResolved() for
// getting a list of all hosts.
func (c *CommandLineQConf) ShowHostGroup(groupName string) (HostGroupConfig, error) {
	out, err := c.RunCommand("-shgrp", groupName)
	if err != nil {
		return HostGroupConfig{}, err
	}
	lines := strings.Split(out, "\n")
	cfg := HostGroupConfig{Name: groupName}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "group_name":
			cfg.Name = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		case "hostlist":
			cfg.Hosts = strings.Split(strings.TrimSpace(
				strings.TrimPrefix(line, fields[0])), " ")
		}
	}
	return cfg, nil
}

// ShowHostGroupResolved shows all hosts in a host group and all sub-groups.
func (c *CommandLineQConf) ShowHostGroupResolved(groupName string) ([]string, error) {
	out, err := c.RunCommand("-shgrp_resolved", groupName)
	if err != nil {
		return nil, err
	}
	return splitWithoutEmptyLines(out, "\n"), nil
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
	if len(users) == 0 {
		return nil
	}
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
	if users == nil {
		return nil
	}
	_, err := c.RunCommand("-ao", strings.Join(users, ","))
	return err
}

// DeleteUserFromOperatorList deletes a list of users from the operator list.
func (c *CommandLineQConf) DeleteUserFromOperatorList(users []string) error {
	if len(users) == 0 {
		return nil
	}
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
	if pe.ControlSlaves == "" {
		pe.ControlSlaves = "FALSE"
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
	file, err := createTempDirWithFileName(pe.Name)
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
	_, err := file.WriteString(fmt.Sprintf("pe_name            %s\n", pe.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("slots             %d\n", pe.Slots))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("user_lists         %s\n", JoinList(pe.UserLists, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xuser_lists        %s\n", JoinList(pe.XUserLists, " ")))
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
	_, err = file.WriteString(fmt.Sprintf("control_slaves     %s\n", pe.ControlSlaves))
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
	_, err = file.WriteString(fmt.Sprintf("ign_sreq_on_mhost  %s\n", MakeBoolCfg(pe.IgnoreSlaveReqestsOnMasterhost)))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("master_forks_slaves  %s\n", MakeBoolCfg(pe.MasterForksSlaves)))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("daemon_forks_slaves  %s\n", MakeBoolCfg(pe.DaemonForksSlaves)))
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
	cfg := ParallelEnvironmentConfig{Name: peName}
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "pe_name":
			cfg.Name = fields[1]
		case "slots":
			cfg.Slots, _ = strconv.Atoi(fields[1])
		case "user_lists":
			cfg.UserLists = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "xuser_lists":
			cfg.XUserLists = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "start_proc_args":
			cfg.StartProcArgs, _ = ParseMultiLineValue(lines, i)
		case "stop_proc_args":
			cfg.StopProcArgs, _ = ParseMultiLineValue(lines, i)
		case "allocation_rule":
			cfg.AllocationRule = fields[1]
		case "control_slaves":
			cfg.ControlSlaves = fields[1]
		case "job_is_first_task":
			cfg.JobIsFirstTask, _ = strconv.ParseBool(fields[1])
		case "urgency_slots":
			cfg.UrgencySlots = fields[1]
		case "accounting_summary":
			cfg.AccountingSummary, _ = strconv.ParseBool(fields[1])
		case "ign_sreq_on_mhost":
			cfg.IgnoreSlaveReqestsOnMasterhost, _ = strconv.ParseBool(fields[1])
		case "master_forks_slaves":
			cfg.MasterForksSlaves, _ = strconv.ParseBool(fields[1])
		case "daemon_forks_slaves":
			cfg.DaemonForksSlaves, _ = strconv.ParseBool(fields[1])
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
	// Nothing todo
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
	_, err = file.WriteString(fmt.Sprintf("acl     %s\n", JoinList(project.ACL, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xacl    %s\n", JoinList(project.XACL, " ")))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Aprj", file.Name())
	return err
}

// DeleteProject deletes the specified projects.
func (c *CommandLineQConf) DeleteProject(projects []string) error {
	if len(projects) == 0 {
		return nil
	}
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
			cfg.ACL = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "xacl":
			cfg.XACL = ParseSpaceSeparatedMultiLineValues(lines, i)
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
	if queue.Name == "" {
		queue.Name = "test"
	}
	if len(queue.HostList) == 0 {
		queue.HostList = []string{"NONE"}
	}
	if len(queue.SeqNo) == 0 || queue.SeqNo[0] == "" {
		queue.SeqNo = []string{"0"}
	}
	if len(queue.LoadThresholds) == 0 ||
		queue.LoadThresholds[0] == "" {
		queue.LoadThresholds = []string{"np_load_avg=1.75"}
	}
	if len(queue.SuspendThresholds) == 0 ||
		queue.SuspendThresholds[0] == "" {
		queue.SuspendThresholds = []string{"NONE"}
	}
	if len(queue.NSuspend) == 0 || queue.NSuspend[0] == "" {
		queue.NSuspend = []string{"0"}
	}
	if len(queue.SuspendInterval) == 0 || queue.SuspendInterval[0] == "" {
		queue.SuspendInterval = []string{"00:05:00"}
	}
	if len(queue.Priority) == 0 || queue.Priority[0] == "" {
		queue.Priority = []string{"0"}
	}
	if len(queue.MinCpuInterval) == 0 || queue.MinCpuInterval[0] == "" {
		queue.MinCpuInterval = []string{"00:05:00"}
	}
	if len(queue.Processors) == 0 || queue.Processors[0] == "" {
		queue.Processors = []string{"UNDEFINED"}
	}
	if len(queue.QType) == 0 {
		queue.QType = []string{"BATCH", "INTERACTIVE"}
	}
	if len(queue.Rerun) == 0 || queue.Rerun[0] == "" {
		queue.Rerun = []string{"FALSE"}
	}
	if len(queue.Slots) == 0 || queue.Slots[0] == "" {
		queue.Slots = []string{"1"}
	}
	if len(queue.TmpDir) == 0 || queue.TmpDir[0] == "" {
		queue.TmpDir = []string{"/tmp"}
	}
	if len(queue.Shell) == 0 || queue.Shell[0] == "" {
		queue.Shell = []string{"/bin/sh"}
	}
	if len(queue.Prolog) == 0 || queue.Prolog[0] == "" {
		queue.Prolog = []string{"NONE"}
	}
	if len(queue.Epilog) == 0 || queue.Epilog[0] == "" {
		queue.Epilog = []string{"NONE"}
	}
	if len(queue.ShellStartMode) == 0 || queue.ShellStartMode[0] == "" {
		queue.ShellStartMode = []string{"unix_behavior"}
	}
	if len(queue.StarterMethod) == 0 || queue.StarterMethod[0] == "" {
		queue.StarterMethod = []string{"NONE"}
	}
	if len(queue.SuspendMethod) == 0 || queue.SuspendMethod[0] == "" {
		queue.SuspendMethod = []string{"NONE"}
	}
	if len(queue.ResumeMethod) == 0 || queue.ResumeMethod[0] == "" {
		queue.ResumeMethod = []string{"NONE"}
	}
	if len(queue.TerminateMethod) == 0 || queue.TerminateMethod[0] == "" {
		queue.TerminateMethod = []string{"NONE"}
	}
	if len(queue.Notify) == 0 || queue.Notify[0] == "" {
		queue.Notify = []string{"00:00:60"}
	}
	if len(queue.Calendar) == 0 || queue.Calendar[0] == "" {
		queue.Calendar = []string{"NONE"}
	}
	if len(queue.InitialState) == 0 || queue.InitialState[0] == "" {
		queue.InitialState = []string{"default"}
	}
	if len(queue.SRt) == 0 || queue.SRt[0] == "" {
		queue.SRt = []string{"INFINITY"}
	}
	if len(queue.HRt) == 0 || queue.HRt[0] == "" {
		queue.HRt = []string{"INFINITY"}
	}
	if len(queue.SCpu) == 0 || queue.SCpu[0] == "" {
		queue.SCpu = []string{"INFINITY"}
	}
	if len(queue.HCpu) == 0 || queue.HCpu[0] == "" {
		queue.HCpu = []string{"INFINITY"}
	}
	if len(queue.SSize) == 0 || queue.SSize[0] == "" {
		queue.SSize = []string{"INFINITY"}
	}
	if len(queue.HSize) == 0 || queue.HSize[0] == "" {
		queue.HSize = []string{"INFINITY"}
	}
	if len(queue.SData) == 0 || queue.SData[0] == "" {
		queue.SData = []string{"INFINITY"}
	}
	if len(queue.HData) == 0 || queue.HData[0] == "" {
		queue.HData = []string{"INFINITY"}
	}
	if len(queue.SStack) == 0 || queue.SStack[0] == "" {
		queue.SStack = []string{"INFINITY"}
	}
	if len(queue.HStack) == 0 || queue.HStack[0] == "" {
		queue.HStack = []string{"INFINITY"}
	}
	if len(queue.SCore) == 0 || queue.SCore[0] == "" {
		queue.SCore = []string{"INFINITY"}
	}
	if len(queue.HCore) == 0 || queue.HCore[0] == "" {
		queue.HCore = []string{"INFINITY"}
	}
	if len(queue.SRss) == 0 || queue.SRss[0] == "" {
		queue.SRss = []string{"INFINITY"}
	}
	if len(queue.HRss) == 0 || queue.HRss[0] == "" {
		queue.HRss = []string{"INFINITY"}
	}
	if len(queue.SVmem) == 0 || queue.SVmem[0] == "" {
		queue.SVmem = []string{"INFINITY"}
	}
	if len(queue.HVmem) == 0 || queue.HVmem[0] == "" {
		queue.HVmem = []string{"INFINITY"}
	}

	// Simplified checks for fields that should be ["NONE"] when empty
	if len(queue.SuspendThresholds) == 0 || queue.SuspendThresholds[0] == "" {
		queue.SuspendThresholds = []string{"NONE"}
	}

	if len(queue.Prolog) == 0 || queue.Prolog[0] == "" {
		queue.Prolog = []string{"NONE"}
	}

	if len(queue.Epilog) == 0 || queue.Epilog[0] == "" {
		queue.Epilog = []string{"NONE"}
	}

	if len(queue.StarterMethod) == 0 || queue.StarterMethod[0] == "" {
		queue.StarterMethod = []string{"NONE"}
	}

	if len(queue.SuspendMethod) == 0 || queue.SuspendMethod[0] == "" {
		queue.SuspendMethod = []string{"NONE"}
	}

	if len(queue.ResumeMethod) == 0 || queue.ResumeMethod[0] == "" {
		queue.ResumeMethod = []string{"NONE"}
	}

	if len(queue.TerminateMethod) == 0 || queue.TerminateMethod[0] == "" {
		queue.TerminateMethod = []string{"NONE"}
	}

	if len(queue.Calendar) == 0 || queue.Calendar[0] == "" {
		queue.Calendar = []string{"NONE"}
	}
}

// AddClusterQueue adds a cluster queue.
func (c *CommandLineQConf) AddClusterQueue(queue ClusterQueueConfig) error {
	SetDefaultQueueValues(&queue)

	file, err := createTempDirWithFileName(queue.Name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("qname             %s\n", queue.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("hostlist          %s\n",
		JoinList(queue.HostList, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("seq_no            %s\n",
		JoinListWithOverrides(queue.SeqNo, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("load_thresholds    %s\n",
		JoinListWithOverrides(queue.LoadThresholds, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_thresholds %s\n",
		JoinListWithOverrides(queue.SuspendThresholds, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("nsuspend          %s\n",
		JoinListWithOverrides(queue.NSuspend, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_interval   %s\n",
		JoinListWithOverrides(queue.SuspendInterval, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("priority          %s\n",
		JoinListWithOverrides(queue.Priority, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("min_cpu_interval    %s\n",
		JoinListWithOverrides(queue.MinCpuInterval, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("processors        %s\n",
		JoinListWithOverrides(queue.Processors, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("qtype             %s\n",
		JoinListWithOverrides(queue.QType, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("ckpt_list          %s\n",
		JoinListWithOverrides(queue.CkptList, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("pe_list            %s\n",
		JoinListWithOverrides(queue.PeList, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("rerun             %s\n",
		JoinListWithOverrides(queue.Rerun, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("slots             %s\n",
		JoinListWithOverrides(queue.Slots, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("tmpdir            %s\n",
		JoinListWithOverrides(queue.TmpDir, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("shell             %s\n",
		JoinListWithOverrides(queue.Shell, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("prolog            %s\n",
		JoinListWithOverrides(queue.Prolog, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("epilog            %s\n",
		JoinListWithOverrides(queue.Epilog, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("shell_start_mode    %s\n",
		JoinListWithOverrides(queue.ShellStartMode, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("starter_method     %s\n",
		JoinListWithOverrides(queue.StarterMethod, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_method     %s\n",
		JoinListWithOverrides(queue.SuspendMethod, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("resume_method      %s\n",
		JoinListWithOverrides(queue.ResumeMethod, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("terminate_method   %s\n",
		JoinListWithOverrides(queue.TerminateMethod, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("notify            %s\n",
		JoinListWithOverrides(queue.Notify, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("owner_list         %s\n",
		JoinListWithOverrides(queue.OwnerList, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("user_lists          %s\n",
		JoinListWithOverrides(queue.UserLists, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xuser_lists         %s\n",
		JoinListWithOverrides(queue.XUserLists, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("subordinate_list   %s\n",
		JoinListWithOverrides(queue.SubordinateList, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("complex_values     %s\n",
		JoinListWithOverrides(queue.ComplexValues, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("projects          %s\n",
		JoinListWithOverrides(queue.Projects, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xprojects         %s\n",
		JoinListWithOverrides(queue.XProjects, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("calendar          %s\n",
		JoinListWithOverrides(queue.Calendar, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("initial_state      %s\n",
		JoinListWithOverrides(queue.InitialState, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_rt               %s\n",
		JoinListWithOverrides(queue.SRt, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_rt               %s\n",
		JoinListWithOverrides(queue.HRt, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_cpu              %s\n",
		JoinListWithOverrides(queue.SCpu, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_cpu              %s\n",
		JoinListWithOverrides(queue.HCpu, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_fsize            %s\n",
		JoinListWithOverrides(queue.SSize, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_fsize            %s\n",
		JoinListWithOverrides(queue.HSize, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_data             %s\n",
		JoinListWithOverrides(queue.SData, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_data             %s\n",
		JoinListWithOverrides(queue.HData, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_stack            %s\n",
		JoinListWithOverrides(queue.SStack, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_stack            %s\n",
		JoinListWithOverrides(queue.HStack, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_core             %s\n",
		JoinListWithOverrides(queue.SCore, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_core             %s\n",
		JoinListWithOverrides(queue.HCore, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_rss              %s\n",
		JoinListWithOverrides(queue.SRss, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_rss              %s\n",
		JoinListWithOverrides(queue.HRss, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_vmem             %s\n",
		JoinListWithOverrides(queue.SVmem, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_vmem             %s\n",
		JoinListWithOverrides(queue.HVmem, ",")))
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
	// switching to single line output
	lines := strings.Split(out, "\n")
	cfg := ClusterQueueConfig{Name: queueName}
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "qname":
			cfg.Name = fields[1]
		case "hostlist":
			cfg.HostList = ParseSpaceSeparatedMultiLineValues(lines, i)
		case "seq_no":
			cfg.SeqNo = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "load_thresholds":
			cfg.LoadThresholds = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "suspend_thresholds":
			cfg.SuspendThresholds = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "nsuspend":
			cfg.NSuspend = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "suspend_interval":
			cfg.SuspendInterval = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "priority":
			cfg.Priority = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "min_cpu_interval":
			cfg.MinCpuInterval = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "processors":
			cfg.Processors = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "qtype":
			cfg.QType = ParseSpaceSeparatedValuesWithOverrides(lines, i)
		case "ckpt_list":
			cfg.CkptList = ParseSpaceSeparatedValuesWithOverrides(lines, i)
		case "pe_list":
			cfg.PeList = ParseSpaceSeparatedValuesWithOverrides(lines, i)
		case "rerun":
			cfg.Rerun = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "slots":
			cfg.Slots = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "tmpdir":
			cfg.TmpDir = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "shell":
			cfg.Shell = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "prolog":
			cfg.Prolog = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "epilog":
			cfg.Epilog = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "shell_start_mode":
			cfg.ShellStartMode = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "starter_method":
			cfg.StarterMethod = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "suspend_method":
			cfg.SuspendMethod = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "resume_method":
			cfg.ResumeMethod = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "terminate_method":
			cfg.TerminateMethod = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "notify":
			cfg.Notify = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "owner_list":
			cfg.OwnerList = ParseSpaceSeparatedValuesWithOverrides(lines, i)
		case "user_lists":
			cfg.UserLists = ParseSpaceSeparatedValuesWithOverrides(lines, i)
		case "xuser_lists":
			cfg.XUserLists = ParseSpaceSeparatedValuesWithOverrides(lines, i)
		case "subordinate_list":
			cfg.SubordinateList = ParseSpaceSeparatedValuesWithOverrides(lines, i)
		case "complex_values":
			cfg.ComplexValues = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "projects":
			cfg.Projects = ParseSpaceSeparatedValuesWithOverrides(lines, i)
		case "xprojects":
			cfg.XProjects = ParseSpaceSeparatedValuesWithOverrides(lines, i)
		case "calendar":
			cfg.Calendar = ParseSpaceSeparatedValuesWithOverrides(lines, i)
		case "initial_state":
			cfg.InitialState = ParseSpaceSeparatedValuesWithOverrides(lines, i)
		case "s_rt":
			cfg.SRt = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "h_rt":
			cfg.HRt = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "s_cpu":
			cfg.SCpu = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "h_cpu":
			cfg.HCpu = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "s_fsize":
			cfg.SSize = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "h_fsize":
			cfg.HSize = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "s_data":
			cfg.SData = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "h_data":
			cfg.HData = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "s_stack":
			cfg.SStack = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "h_stack":
			cfg.HStack = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "s_core":
			cfg.SCore = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "h_core":
			cfg.HCore = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "s_rss":
			cfg.SRss = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "h_rss":
			cfg.HRss = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "s_vmem":
			cfg.SVmem = ParseCommaSeparatedValuesWithOverrides(lines, i)
		case "h_vmem":
			cfg.HVmem = ParseCommaSeparatedValuesWithOverrides(lines, i)
		}
	}

	// THIS IS THE CRITICAL PART
	// Make sure all nil fields are properly converted to ["NONE"]
	SetDefaultQueueValues(&cfg)

	return cfg, nil
}

// ShowClusterQueues shows all cluster queues.
func (c *CommandLineQConf) ShowClusterQueues() ([]string, error) {
	output, err := c.RunCommand("-sql")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
		return nil, err
	}
	return splitWithoutEmptyLines(output, "\n"), nil
}

// AddSubmitHosts adds a list of submit hosts.
func (c *CommandLineQConf) AddSubmitHosts(hostnames []string) error {
	if hostnames == nil {
		return nil
	}
	_, err := c.RunCommand("-as", strings.Join(hostnames, ","))
	return err
}

// DeleteSubmitHost deletes a list of submit hosts.
func (c *CommandLineQConf) DeleteSubmitHost(hostnames []string) error {
	if hostnames == nil {
		return nil
	}
	_, err := c.RunCommand("-ds", strings.Join(hostnames, ","))
	return err
}

// ShowSubmitHosts shows all submit hosts.
func (c *CommandLineQConf) ShowSubmitHosts() ([]string, error) {
	output, err := c.RunCommand("-ss")
	if err != nil {
		if strings.Contains(output, "no") &&
			strings.Contains(output, "defined") {
			return []string{}, nil
		}
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
	for i, line := range lines {
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
			cfg.Entries = ParseCommaSeparatedMultiLineValues(lines, i)
		}
	}
	return cfg, nil
}

func SetDefaultUserSetListConfig(u *UserSetListConfig) {
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
	_, err = file.WriteString(fmt.Sprintf("entries           %s\n", JoinList(u.Entries, " ")))
	if err != nil {
		return err
	}
	file.Close()
	_, err = c.RunCommand("-Au", file.Name())
	return err
}

// AddUser adds a new user.
func (c *CommandLineQConf) AddUser(userConfig UserConfig) error {
	if userConfig.DefaultProject == "" {
		userConfig.DefaultProject = "NONE"
	}
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
	if len(users) == 0 {
		return nil
	}
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
func (c *CommandLineQConf) ClearShareTreeUsage() error {
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
	if centries == nil {
		return nil
	}
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

// ModifyComplexEntry modifies a complex entry.
func (c *CommandLineQConf) ModifyComplexEntry(complexName string, cfg ComplexEntryConfig) error {
	file, err := createTempDirWithFileName(complexName)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(fmt.Sprintf("name           %s\n", cfg.Name))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("shortcut       %s\n", cfg.Shortcut))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("type           %s\n", cfg.Type))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("relop       %s\n", cfg.Relop))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("requestable  %s\n", cfg.Requestable))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("consumable   %s\n", cfg.Consumable))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("default      %s\n", cfg.Default))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("urgency      %d\n", cfg.Urgency))
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Mce", file.Name())
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
	_, err = file.WriteString(fmt.Sprintf("ckpt_command  %s\n", cfg.CheckpointCommand))
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
	_, err = file.WriteString(fmt.Sprintf("ckpt_dir        %s\n", cfg.CheckpointDir))
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

	// Write pointer fields if they're not nil
	if cfg.ExecdSpoolDir != nil {
		if _, err = file.WriteString(fmt.Sprintf("execd_spool_dir %s\n", *cfg.ExecdSpoolDir)); err != nil {
			return err
		}
	}
	if cfg.Mailer != nil {
		if _, err = file.WriteString(fmt.Sprintf("mailer %s\n", *cfg.Mailer)); err != nil {
			return err
		}
	}
	if cfg.Xterm != nil {
		if _, err = file.WriteString(fmt.Sprintf("xterm %s\n", *cfg.Xterm)); err != nil {
			return err
		}
	}

	// Handle slice fields
	for _, sensor := range cfg.LoadSensors {
		if _, err = file.WriteString(fmt.Sprintf("load_sensor %s\n", sensor)); err != nil {
			return err
		}
	}

	if cfg.Prolog != nil {
		if _, err = file.WriteString(fmt.Sprintf("prolog %s\n", *cfg.Prolog)); err != nil {
			return err
		}
	}
	if cfg.Epilog != nil {
		if _, err = file.WriteString(fmt.Sprintf("epilog %s\n", *cfg.Epilog)); err != nil {
			return err
		}
	}
	if cfg.ShellStartMode != nil {
		if _, err = file.WriteString(fmt.Sprintf("shell_start_mode %s\n", *cfg.ShellStartMode)); err != nil {
			return err
		}
	}

	for _, shell := range cfg.LoginShells {
		if _, err = file.WriteString(fmt.Sprintf("login_shells %s\n", shell)); err != nil {
			return err
		}
	}

	if cfg.LoadReportTime != nil {
		if _, err = file.WriteString(fmt.Sprintf("load_report_time %s\n", *cfg.LoadReportTime)); err != nil {
			return err
		}
	}
	if cfg.SetTokenCmd != nil {
		if _, err = file.WriteString(fmt.Sprintf("set_token_cmd %s\n", *cfg.SetTokenCmd)); err != nil {
			return err
		}
	}
	if cfg.PagCmd != nil {
		if _, err = file.WriteString(fmt.Sprintf("pag_cmd %s\n", *cfg.PagCmd)); err != nil {
			return err
		}
	}
	if cfg.TokenExtendTime != nil {
		if _, err = file.WriteString(fmt.Sprintf("token_extend_time %s\n", *cfg.TokenExtendTime)); err != nil {
			return err
		}
	}
	if cfg.ShepherdCmd != nil {
		if _, err = file.WriteString(fmt.Sprintf("shepherd_cmd %s\n", *cfg.ShepherdCmd)); err != nil {
			return err
		}
	}

	for _, param := range cfg.ExecdParams {
		if _, err = file.WriteString(fmt.Sprintf("execd_params %s\n", param)); err != nil {
			return err
		}
	}

	for _, param := range cfg.ReportingParams {
		if _, err = file.WriteString(fmt.Sprintf("reporting_params %s\n", param)); err != nil {
			return err
		}
	}

	for _, gid := range cfg.GidRange {
		if _, err = file.WriteString(fmt.Sprintf("gid_range %s\n", gid)); err != nil {
			return err
		}
	}

	if cfg.QloginDaemon != nil {
		if _, err = file.WriteString(fmt.Sprintf("qlogin_daemon %s\n", *cfg.QloginDaemon)); err != nil {
			return err
		}
	}
	if cfg.QloginCommand != nil {
		if _, err = file.WriteString(fmt.Sprintf("qlogin_command %s\n", *cfg.QloginCommand)); err != nil {
			return err
		}
	}
	if cfg.RshDaemon != nil {
		if _, err = file.WriteString(fmt.Sprintf("rsh_daemon %s\n", *cfg.RshDaemon)); err != nil {
			return err
		}
	}
	if cfg.RshCommand != nil {
		if _, err = file.WriteString(fmt.Sprintf("rsh_command %s\n", *cfg.RshCommand)); err != nil {
			return err
		}
	}
	if cfg.RloginDaemon != nil {
		if _, err = file.WriteString(fmt.Sprintf("rlogin_daemon %s\n", *cfg.RloginDaemon)); err != nil {
			return err
		}
	}
	if cfg.RloginCommand != nil {
		if _, err = file.WriteString(fmt.Sprintf("rlogin_command %s\n", *cfg.RloginCommand)); err != nil {
			return err
		}
	}
	if cfg.RescheduleUnknown != nil {
		if _, err = file.WriteString(fmt.Sprintf("reschedule_unknown %s\n", *cfg.RescheduleUnknown)); err != nil {
			return err
		}
	}
	if cfg.LibJvmPath != nil {
		if _, err = file.WriteString(fmt.Sprintf("libjvm_path %s\n", *cfg.LibJvmPath)); err != nil {
			return err
		}
	}
	if cfg.AdditionalJvmArgs != nil {
		if _, err = file.WriteString(fmt.Sprintf("additional_jvm_args %s\n", *cfg.AdditionalJvmArgs)); err != nil {
			return err
		}
	}

	file.Close()

	_, err = c.RunCommand("-Mconf", file.Name())
	return err
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
		// if type is []string, join the values either
		// comma separated or space separated depending on fieldName
		if reflect.TypeOf(fieldValue).Kind() == reflect.Slice {
			if len(fieldValue.([]string)) == 0 {
				fieldValue = []string{"NONE"}
			}
			switch fieldName {
			case "complex_values", "login_shells", "qmaster_params",
				"execd_params", "load_sensor", "gid_range", "jsv_allowed_mod":
				fieldValue = strings.Join(fieldValue.([]string), ",")
			case "user_lists", "xuser_lists", "projects", "xprojects", "reporting_params":
				fieldValue = strings.Join(fieldValue.([]string), " ")
			default:
				return fmt.Errorf("unsupported slice type: %s", fieldName)
			}
		}

		// an empty string should be written as "NONE"
		if reflect.TypeOf(fieldValue).Kind() == reflect.String {
			if fieldValue.(string) == "" {
				fieldValue = "NONE"
			}
		}
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
	// Nothing to do here.
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
	_, err = file.WriteString(fmt.Sprintf("load_scaling     %s\n",
		JoinStringFloatMap(cfg.LoadScaling, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("complex_values   %s\n",
		JoinStringStringMap(cfg.ComplexValues, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("user_lists       %s\n", JoinList(cfg.UserLists, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xuser_lists      %s\n", JoinList(cfg.XUserLists, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("projects         %s\n", JoinList(cfg.Projects, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xprojects        %s\n", JoinList(cfg.XProjects, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("usage_scaling    %s\n",
		JoinStringFloatMap(cfg.UsageScaling, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("report_variables %s\n", JoinList(cfg.ReportVariables, ",")))
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

	_, err = file.WriteString(fmt.Sprintf("group_name %s\n", hostGroupName))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("hostlist %s\n", JoinList(cfg.Hosts, " ")))
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
	_, err = file.WriteString(fmt.Sprintf("acl     %s\n", JoinList(cfg.ACL, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xacl    %s\n", JoinList(cfg.XACL, " ")))
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

	_, err = file.WriteString(fmt.Sprintf("hostlist          %s\n", JoinList(cfg.HostList, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("seq_no            %s\n", JoinListWithOverrides(cfg.SeqNo, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("load_thresholds    %s\n", JoinListWithOverrides(cfg.LoadThresholds, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_thresholds %s\n", JoinListWithOverrides(cfg.SuspendThresholds, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("nsuspend          %s\n", JoinListWithOverrides(cfg.NSuspend, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_interval   %s\n", JoinListWithOverrides(cfg.SuspendInterval, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("priority          %s\n", JoinListWithOverrides(cfg.Priority, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("min_cpu_interval    %s\n", JoinListWithOverrides(cfg.MinCpuInterval, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("processors        %s\n", JoinListWithOverrides(cfg.Processors, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("qtype             %s\n", JoinListWithOverrides(cfg.QType, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("ckpt_list          %s\n", JoinListWithOverrides(cfg.CkptList, " ")))
	if err != nil {
		return err
	}
	// pe1,p2 vs p1,[host=p2]
	_, err = file.WriteString(fmt.Sprintf("pe_list            %s\n", JoinListWithOverrides(cfg.PeList, " ")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("rerun             %s\n", JoinListWithOverrides(cfg.Rerun, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("slots             %s\n", JoinListWithOverrides(cfg.Slots, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("tmpdir            %s\n", JoinListWithOverrides(cfg.TmpDir, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("shell             %s\n", JoinListWithOverrides(cfg.Shell, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("prolog            %s\n", JoinListWithOverrides(cfg.Prolog, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("epilog            %s\n", JoinListWithOverrides(cfg.Epilog, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("shell_start_mode    %s\n", JoinListWithOverrides(cfg.ShellStartMode, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("starter_method     %s\n", JoinListWithOverrides(cfg.StarterMethod, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("suspend_method     %s\n", JoinListWithOverrides(cfg.SuspendMethod, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("resume_method      %s\n", JoinListWithOverrides(cfg.ResumeMethod, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("terminate_method   %s\n", JoinListWithOverrides(cfg.TerminateMethod, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("notify            %s\n", JoinListWithOverrides(cfg.Notify, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("owner_list         %s\n",
		JoinListWithOverrides(cfg.OwnerList, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("user_lists          %s\n",
		JoinListWithOverrides(cfg.UserLists, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xuser_lists         %s\n",
		JoinListWithOverrides(cfg.XUserLists, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("subordinate_list   %s\n",
		JoinListWithOverrides(cfg.SubordinateList, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("complex_values     %s\n",
		JoinListWithOverrides(cfg.ComplexValues, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("projects          %s\n",
		JoinListWithOverrides(cfg.Projects, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("xprojects         %s\n",
		JoinListWithOverrides(cfg.XProjects, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("calendar          %s\n",
		JoinListWithOverrides(cfg.Calendar, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("initial_state      %s\n",
		JoinListWithOverrides(cfg.InitialState, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_rt               %s\n",
		JoinListWithOverrides(cfg.SRt, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_rt               %s\n",
		JoinListWithOverrides(cfg.HRt, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_cpu              %s\n",
		JoinListWithOverrides(cfg.SCpu, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_cpu              %s\n",
		JoinListWithOverrides(cfg.HCpu, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_fsize            %s\n",
		JoinListWithOverrides(cfg.SSize, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_fsize            %s\n",
		JoinListWithOverrides(cfg.HSize, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_data             %s\n",
		JoinListWithOverrides(cfg.SData, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_data             %s\n",
		JoinListWithOverrides(cfg.HData, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_stack            %s\n",
		JoinListWithOverrides(cfg.SStack, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_stack            %s\n",
		JoinListWithOverrides(cfg.HStack, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_core             %s\n",
		JoinListWithOverrides(cfg.SCore, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_core             %s\n",
		JoinListWithOverrides(cfg.HCore, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_rss              %s\n",
		JoinListWithOverrides(cfg.SRss, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_rss              %s\n",
		JoinListWithOverrides(cfg.HRss, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("s_vmem             %s\n",
		JoinListWithOverrides(cfg.SVmem, ",")))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("h_vmem             %s\n",
		JoinListWithOverrides(cfg.HVmem, ",")))
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
	_, err = file.WriteString(fmt.Sprintf("entries %s\n",
		JoinList(cfg.Entries, " ")))
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

// ShowSchedulerConfiguration shows the scheduler configuration.
func (c *CommandLineQConf) ShowSchedulerConfiguration() (*SchedulerConfig, error) {
	out, err := c.RunCommand("-ssconf")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	cfg := SchedulerConfig{}
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value := fields[1]

		switch fields[0] {
		case "algorithm":
			cfg.Algorithm = value
		case "schedule_interval":
			cfg.ScheduleInterval = value
		case "maxujobs":
			cfg.MaxUJobs, _ = strconv.Atoi(value)
		case "queue_sort_method":
			cfg.QueueSortMethod = value
		case "job_load_adjustments":
			cfg.JobLoadAdjustments = ParseCommaSeparatedMultiLineValues(lines, i)
		case "load_adjustment_decay_time":
			cfg.LoadAdjustmentDecayTime = value
		case "load_formula":
			cfg.LoadFormula = value
		case "schedd_job_info":
			cfg.ScheddJobInfo = value
		case "flush_submit_sec":
			cfg.FlushSubmitSec, _ = strconv.Atoi(value)
		case "flush_finish_sec":
			cfg.FlushFinishSec, _ = strconv.Atoi(value)
		case "params":
			cfg.Params = ParseSpaceAndCommaSeparatedMultiLineValues(lines, i)
		case "reprioritize_interval":
			cfg.ReprioritizeInterval = value
		case "halftime":
			cfg.Halftime, _ = strconv.Atoi(value)
		case "usage_weight_list":
			cfg.UsageWeightList = ParseCommaSeparatedMultiLineValues(lines, i)
		case "compensation_factor":
			cfg.CompensationFactor, _ = strconv.ParseFloat(value, 64)
		case "weight_user":
			cfg.WeightUser, _ = strconv.ParseFloat(value, 64)
		case "weight_project":
			cfg.WeightProject, _ = strconv.ParseFloat(value, 64)
		case "weight_department":
			cfg.WeightDepartment, _ = strconv.ParseFloat(value, 64)
		case "weight_job":
			cfg.WeightJob, _ = strconv.ParseFloat(value, 64)
		case "weight_tickets_functional":
			cfg.WeightTicketsFunctional, _ = strconv.Atoi(value)
		case "weight_tickets_share":
			cfg.WeightTicketsShare, _ = strconv.Atoi(value)
		case "share_override_tickets":
			cfg.ShareOverrideTickets, _ = strconv.ParseBool(value)
		case "share_functional_shares":
			cfg.ShareFunctionalShares, _ = strconv.ParseBool(value)
		case "max_functional_jobs_to_schedule":
			cfg.MaxFunctionalJobsToSchedule, _ = strconv.Atoi(value)
		case "report_pjob_tickets":
			cfg.ReportPJobTickets, _ = strconv.ParseBool(value)
		case "max_pending_tasks_per_job":
			cfg.MaxPendingTasksPerJob, _ = strconv.Atoi(value)
		case "halflife_decay_list":
			cfg.HalflifeDecayList = ParseCommaSeparatedMultiLineValues(lines, i)
		case "policy_hierarchy":
			cfg.PolicyHierarchy = value
		case "weight_ticket":
			cfg.WeightTicket, _ = strconv.ParseFloat(value, 64)
		case "weight_waiting_time":
			cfg.WeightWaitingTime, _ = strconv.ParseFloat(value, 64)
		case "weight_deadline":
			cfg.WeightDeadline, _ = strconv.ParseFloat(value, 64)
		case "weight_urgency":
			cfg.WeightUrgency, _ = strconv.ParseFloat(value, 64)
		case "weight_priority":
			cfg.WeightPriority, _ = strconv.ParseFloat(value, 64)
		case "max_reservation":
			cfg.MaxReservation, _ = strconv.Atoi(value)
		case "default_duration":
			cfg.DefaultDuration = value
		}
	}
	return &cfg, nil
}

// ModifySchedulerConfig modifies the scheduler configuration.
func (c *CommandLineQConf) ModifySchedulerConfig(cfg SchedulerConfig) error {

	// set defaults
	if cfg.LoadAdjustmentDecayTime == "" {
		cfg.LoadAdjustmentDecayTime = "00:00:00"
	}

	if cfg.ScheduleInterval == "" {
		cfg.ScheduleInterval = "00:00:00"
	}

	if cfg.ScheddJobInfo == "" {
		cfg.ScheddJobInfo = "false"
	}

	if cfg.DefaultDuration == "" {
		cfg.DefaultDuration = "00:00:00"
	}

	if cfg.ReprioritizeInterval == "" {
		cfg.ReprioritizeInterval = "00:00:00"
	}

	if cfg.MaxPendingTasksPerJob == 0 {
		// must be > 0
		cfg.MaxPendingTasksPerJob = 50
	}

	if cfg.Algorithm == "" {
		cfg.Algorithm = "default"
	}

	if cfg.LoadFormula == "" {
		cfg.LoadFormula = "np_load_avg"
	}

	file, err := createTempDirWithFileName("scheduler")
	if err != nil {
		return err
	}
	//defer os.RemoveAll(filepath.Dir(file.Name()))

	v := reflect.ValueOf(cfg)
	typeOfS := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldName := typeOfS.Field(i).Tag.Get("json")
		fieldValue := v.Field(i).Interface()
		// if type is []string, join the values either
		// comma separated or space separated depending on fieldName
		if reflect.TypeOf(fieldValue).Kind() == reflect.Slice {
			if len(fieldValue.([]string)) == 0 {
				fieldValue = []string{"NONE"}
			}
			switch fieldName {
			case "job_load_adjustments", "halflife_decay_list", "usage_weight_list", "params":
				fieldValue = strings.Join(fieldValue.([]string), ",")
			default:
				return fmt.Errorf("unsupported slice type: %s", fieldName)
			}
		}

		// for booleans, write "TRUE" or "FALSE"
		if reflect.TypeOf(fieldValue).Kind() == reflect.Bool {
			if fieldValue.(bool) {
				fieldValue = "TRUE"
			} else {
				fieldValue = "FALSE"
			}
		}

		// for float64, write the value with 6 decimal places
		if reflect.TypeOf(fieldValue).Kind() == reflect.Float64 {
			fieldValue = fmt.Sprintf("%.6f", fieldValue)
		}

		// an empty string should be written as "NONE"
		if reflect.TypeOf(fieldValue).Kind() == reflect.String {
			if fieldValue.(string) == "" {
				fieldValue = "NONE"
			}
		}

		_, err = file.WriteString(fmt.Sprintf("%s %v\n", fieldName, fieldValue))
		if err != nil {
			return err
		}
	}
	file.Close()

	_, err = c.RunCommand("-Msconf", file.Name())
	return err
}

// ShowShareTree retrieves the entire share tree structure
func (c *CommandLineQConf) ShowShareTree() (string, error) {
	stree, err := c.RunCommand("-sstree")
	if err != nil {
		if strings.Contains(err.Error(), "no sharetree") {
			return "", fmt.Errorf("no sharetree defined")
		}
		return "", err
	}
	return stree, nil
}

// ModifyShareTreeNodes modifies the share of specified nodes in the share tree
func (c *CommandLineQConf) ModifyShareTreeNodes(nodeShareList []ShareTreeNode) error {
	var nodeShares []string
	for _, node := range nodeShareList {
		nodeShares = append(nodeShares, fmt.Sprintf("%s=%d", node.Node, node.Share))
	}
	_, err := c.RunCommand("-mstnode", strings.Join(nodeShares, ","))
	return err
}

// DeleteShareTreeNodes removes specified nodes from the share tree
func (c *CommandLineQConf) DeleteShareTreeNodes(nodeList []string) error {
	_, err := c.RunCommand(append([]string{"-dstnode"}, nodeList...)...)
	return err
}

// AddShareTreeNode adds a new node to the share tree. The node must be a full
// path, like /P1.
func (c *CommandLineQConf) AddShareTreeNode(node ShareTreeNode) error {
	_, err := c.RunCommand("-astnode", fmt.Sprintf("%s=%d", node.Node, node.Share))
	return err
}

// ShowShareTreeNodes retrieves information about specified nodes or all
// nodes in the share tree. The node contains the full path, like /P1.
func (c *CommandLineQConf) ShowShareTreeNodes(nodeList []string) ([]ShareTreeNode, error) {
	// if nodeList is empty, show all nodes
	args := []string{"-sstnode"}
	if len(nodeList) == 0 {
		args = append(args, "/")
	} else {
		args = append(args, nodeList...)
	}
	output, err := c.RunCommand(args...)
	if err != nil {
		return nil, err
	}
	return parseShareTreeNodes(output), nil
}

// ModifyShareTree modifies the entire share tree configuration. If the
// shareTreeConfig is empty, the share tree is deleted.
// A share tree has typically the following format:
// id=0
// name=Root
// type=0
// shares=1
// childnodes=1,2,3
// where childnodes is a comma separated list of child nodes.
func (c *CommandLineQConf) ModifyShareTree(shareTreeConfig string) error {
	// if shareTreeConfig is empty, delete the share tree
	if shareTreeConfig == "" {
		// qconf -dstree
		_, err := c.RunCommand("-dstree")
		if err != nil {
			// Ignore "sharetree does not exist"
			if !strings.Contains(err.Error(), "sharetree does not exist") {
				return err
			}
		}
		return nil
	}

	file, err := createTempDirWithFileName("sharetree")
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	_, err = file.WriteString(shareTreeConfig)
	if err != nil {
		return err
	}
	file.Close()

	_, err = c.RunCommand("-Mstree", file.Name())
	return err
}

// DeleteShareTree deletes the share tree
func (c *CommandLineQConf) DeleteShareTree() error {
	// check if share tree exists
	_, err := c.ShowShareTree()
	if err != nil {
		if !strings.Contains(err.Error(), "no sharetree") {
			return err
		}
		return nil
	}
	_, err = c.RunCommand("-dstree")
	return err
}

// Helper function to parse the output of ShowShareTreeNodes,
// like:
// /=1
// /default=10
// /P2=11
// /P1=11
func parseShareTreeNodes(output string) []ShareTreeNode {
	lines := strings.Split(output, "\n")
	var nodes []ShareTreeNode
	for _, line := range lines {
		parts := strings.Split(line, "=")
		if len(parts) >= 2 {
			share, _ := strconv.Atoi(parts[1])
			// convert node path to node name, like /P1 to P1
			// always after the last /
			/*node := parts[0]
			slashIndex := strings.LastIndex(node, "/")
			if slashIndex != -1 && slashIndex < len(node)-1 {
				node = node[slashIndex+1:]
			}*/
			nodes = append(nodes, ShareTreeNode{
				Node:  parts[0],
				Share: share,
			})
		}
	}
	return nodes
}
