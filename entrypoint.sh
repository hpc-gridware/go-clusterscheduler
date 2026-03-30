#!/bin/bash

#___INFO__MARK_BEGIN_NEW__
###########################################################################
#
#  Copyright 2025-2026 HPC-Gridware GmbH
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
#
###########################################################################
#___INFO__MARK_END_NEW__

set -e

SGE_ROOT="/opt/ocs"

if [ -f "${SGE_ROOT}/default/common/settings.sh" ]; then
    echo "OCS already installed. Starting daemons..."
    source "${SGE_ROOT}/default/common/settings.sh"
    "${SGE_ROOT}/default/common/sgemaster" || true
    "${SGE_ROOT}/default/common/sgeexecd" || true
else
    echo "Installing OCS ${OCS_VERSION:-9.1.0}..."
    cd /tmp
    OCS_VERSION="${OCS_VERSION:-9.1.0}" /opt/helpers/ocs.sh
    if [ -f "${SGE_ROOT}/default/common/settings.sh" ]; then
        source "${SGE_ROOT}/default/common/settings.sh"
    fi
fi

echo "Waiting for cluster to be ready..."
for i in $(seq 1 30); do
    if qconf -sel 2>/dev/null | grep -q "$(hostname)"; then
        echo "Cluster ready (exec host $(hostname) registered)."
        break
    fi
    sleep 1
done

if ! grep -q "${SGE_ROOT}/default/common/settings.sh" /root/.bashrc 2>/dev/null; then
    echo "source ${SGE_ROOT}/default/common/settings.sh" >> /root/.bashrc
fi

export LD_LIBRARY_PATH="${SGE_ROOT}/lib/lx-amd64${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"

cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler 2>/dev/null || true

if [ $# -gt 0 ]; then
    exec "$@"
else
    exec /bin/bash
fi
