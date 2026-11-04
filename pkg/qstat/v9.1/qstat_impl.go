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

package qstat

import (
	"context"

	v90 "github.com/hpc-gridware/go-clusterscheduler/pkg/qstat/v9.0"
)

// CommandLineQStatConfig configures the qstat client; it is the 9.0 one.
type CommandLineQStatConfig = v90.CommandLineQStatConfig

// QStatImpl is the 9.1 qstat client. It embeds the 9.0 client, whose
// command and streaming surface is unchanged in 9.1, and adds the methods
// whose answer differs in 9.1: the "qstat -j" text parser and the
// "qstat -j -xml" job detail decoder.
//
// The zero value is not usable: the embedded 9.0 client is nil, so every
// promoted method panics. Construct it with NewCommandLineQstat. It is a
// distinct type since the unreleased change described in the package
// documentation, so a *v9.0.QStatImpl no longer assigns to it.
type QStatImpl struct {
	*v90.QStatImpl
}

var _ QStat = (*QStatImpl)(nil)

// NewCommandLineQstat creates a 9.1 qstat client around the 9.0 one.
func NewCommandLineQstat(config CommandLineQStatConfig) (*QStatImpl, error) {
	base, err := v90.NewCommandLineQstat(config)
	if err != nil {
		return nil, err
	}
	return &QStatImpl{QStatImpl: base}, nil
}

// SchedulerJobInfoContext runs "qstat -j <ids>" under ctx and parses the
// answer with this package's 9.1 parser, shadowing the embedded 9.0 method
// of the same name. ErrNoSuchJob is returned when every id is unknown.
func (q *QStatImpl) SchedulerJobInfoContext(ctx context.Context, env []string, ids ...string) ([]SchedulerJobInfo, error) {
	return SchedulerJobInfoContext(ctx, env, q.QStatImpl, ids...)
}
