/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2025 HPC-Gridware GmbH
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
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

// CommandLineQConf wraps the core implementation and overrides methods
// that return or accept v9.1-specific types (GlobalConfig).
// All other methods are promoted from the embedded core.CommandLineQConf.
type CommandLineQConf struct {
	*core.CommandLineQConf
}

// NewCommandLineQConf creates a new v9.1 CommandLineQConf instance.
func NewCommandLineQConf(config CommandLineQConfConfig) (*CommandLineQConf, error) {
	c, err := core.NewCommandLineQConf(config)
	if err != nil {
		return nil, err
	}
	return &CommandLineQConf{CommandLineQConf: c}, nil
}

// ShowGlobalConfiguration returns the global configuration with v9.1-specific
// fields parsed. It reuses the core parser for base fields and adds a second
// pass for v9.1-only fields.
func (c *CommandLineQConf) ShowGlobalConfiguration() (*GlobalConfig, error) {
	out, err := c.RunCommand("-sconf", "global")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")

	// Parse base (v9.0/core) fields.
	baseCfg := core.ParseGlobalConfigFromLines(lines)

	cfg := &GlobalConfig{
		GlobalConfig: baseCfg,
	}

	// Parse v9.1-specific fields.
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "jsv_params":
			cfg.JsvParams = fields[1]
		case "topology_file":
			cfg.TopologyFile = fields[1]
		case "mail_tag":
			cfg.MailTag = fields[1]
		case "gdi_request_limits":
			cfg.GDIRequestLimits = fields[1]
		case "binding_params":
			cfg.BindingParams = core.ParseIntoStringStringMap(fields[1], ",")
		}
	}

	return cfg, nil
}

// ModifyGlobalConfig modifies the global configuration with v9.1-specific
// fields included. It writes both the embedded core fields and v9.1 fields
// to the configuration file.
func (c *CommandLineQConf) ModifyGlobalConfig(cfg GlobalConfig) error {
	file, err := core.CreateTempDirWithFileName("global")
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(file.Name()))

	// Write embedded core GlobalConfig fields.
	v := reflect.ValueOf(cfg.GlobalConfig)
	typeOfS := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldName := typeOfS.Field(i).Tag.Get("json")
		fieldValue := v.Field(i).Interface()

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

	// Write v9.1-specific fields.
	writeField := func(name, value string) error {
		if value == "" {
			value = "NONE"
		}
		_, err := file.WriteString(fmt.Sprintf("%s %s\n", name, value))
		return err
	}

	if err := writeField("jsv_params", cfg.JsvParams); err != nil {
		return err
	}
	if err := writeField("topology_file", cfg.TopologyFile); err != nil {
		return err
	}
	if err := writeField("mail_tag", cfg.MailTag); err != nil {
		return err
	}
	if err := writeField("gdi_request_limits", cfg.GDIRequestLimits); err != nil {
		return err
	}

	// Write binding_params as comma-separated key=value pairs.
	bindingStr := core.JoinStringStringMap(cfg.BindingParams, ",")
	if err := writeField("binding_params", bindingStr); err != nil {
		return err
	}

	file.Close()

	_, err = c.RunCommand("-Mconf", file.Name())
	return err
}

// GetClusterConfiguration returns the full cluster configuration with
// v9.1 GlobalConfig.
func (c *CommandLineQConf) GetClusterConfiguration() (ClusterConfig, error) {
	// Get the core cluster config.
	coreCfg, err := c.CommandLineQConf.GetClusterConfiguration()
	if err != nil {
		return ClusterConfig{}, err
	}

	// Convert to v9.1 ClusterConfig.
	cfg := ClusterConfig{
		ClusterEnvironment:   coreCfg.ClusterEnvironment,
		SchedulerConfig:      coreCfg.SchedulerConfig,
		Calendars:            coreCfg.Calendars,
		ComplexEntries:       coreCfg.ComplexEntries,
		CkptInterfaces:       coreCfg.CkptInterfaces,
		HostConfigurations:   coreCfg.HostConfigurations,
		ExecHosts:            coreCfg.ExecHosts,
		AdminHosts:           coreCfg.AdminHosts,
		SubmitHosts:          coreCfg.SubmitHosts,
		HostGroups:           coreCfg.HostGroups,
		ResourceQuotaSets:    coreCfg.ResourceQuotaSets,
		Managers:             coreCfg.Managers,
		Operators:            coreCfg.Operators,
		ParallelEnvironments: coreCfg.ParallelEnvironments,
		Projects:             coreCfg.Projects,
		Users:                coreCfg.Users,
		ClusterQueues:        coreCfg.ClusterQueues,
		UserSetLists:         coreCfg.UserSetLists,
	}

	// Replace GlobalConfig with v9.1 version.
	if coreCfg.GlobalConfig != nil {
		v91Global, err := c.ShowGlobalConfiguration()
		if err != nil {
			return ClusterConfig{}, err
		}
		cfg.GlobalConfig = v91Global
	}

	return cfg, nil
}

// ApplyClusterConfiguration applies a v9.1 cluster configuration.
func (c *CommandLineQConf) ApplyClusterConfiguration(cc ClusterConfig) error {
	// Convert to core ClusterConfig for the base apply.
	coreCfg := core.ClusterConfig{
		ClusterEnvironment:   cc.ClusterEnvironment,
		SchedulerConfig:      cc.SchedulerConfig,
		Calendars:            cc.Calendars,
		ComplexEntries:       cc.ComplexEntries,
		CkptInterfaces:       cc.CkptInterfaces,
		HostConfigurations:   cc.HostConfigurations,
		ExecHosts:            cc.ExecHosts,
		AdminHosts:           cc.AdminHosts,
		SubmitHosts:          cc.SubmitHosts,
		HostGroups:           cc.HostGroups,
		ResourceQuotaSets:    cc.ResourceQuotaSets,
		Managers:             cc.Managers,
		Operators:            cc.Operators,
		ParallelEnvironments: cc.ParallelEnvironments,
		Projects:             cc.Projects,
		Users:                cc.Users,
		ClusterQueues:        cc.ClusterQueues,
		UserSetLists:         cc.UserSetLists,
	}

	// Handle GlobalConfig separately with v9.1 fields.
	if cc.GlobalConfig != nil {
		coreCfg.GlobalConfig = &cc.GlobalConfig.GlobalConfig
	}

	err := c.CommandLineQConf.ApplyClusterConfiguration(coreCfg)
	if err != nil {
		return err
	}

	// Apply v9.1-specific GlobalConfig fields.
	if cc.GlobalConfig != nil {
		err = c.ModifyGlobalConfig(*cc.GlobalConfig)
		if err != nil {
			return err
		}
	}

	return nil
}
