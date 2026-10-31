/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2026 HPC-Gridware GmbH
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

package qquota

import (
	"io"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/qquota/core"
)

// The streaming XML surface is the core one; see core/stream.go for what
// qquota prints and leaves out.

// QuotaStreamer is the bounded-memory, context-aware qquota client.
type QuotaStreamer = core.QuotaStreamer

// QQuotaImpl runs qquota.
type QQuotaImpl = core.QQuotaImpl

// CommandLineQQuotaConfig is the configuration for the QQuotaImpl.
type CommandLineQQuotaConfig = core.CommandLineQQuotaConfig

// QuotaRuleXML is one <qquota_rule> element.
type QuotaRuleXML = core.QuotaRuleXML

// LimitXML is one <limit> of a rule.
type LimitXML = core.LimitXML

// QuotaStats describes one streamed document.
type QuotaStats = core.QuotaStats

// ErrIncompleteQuotaDocument is returned when qquota's output did not end
// with the root close tag.
var ErrIncompleteQuotaDocument = core.ErrIncompleteQuotaDocument

// NewCommandLineQQuota creates a new QQuotaImpl.
func NewCommandLineQQuota(config CommandLineQQuotaConfig) (*QQuotaImpl, error) {
	return core.NewCommandLineQQuota(config)
}

// DecodeQuotasXML reads a "qquota -xml" document and calls fn once per rule.
func DecodeQuotasXML(r io.Reader, fn func(QuotaRuleXML) error) (QuotaStats, error) {
	return core.DecodeQuotasXML(r, fn)
}
