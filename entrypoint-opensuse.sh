#!/bin/bash

#___INFO__MARK_BEGIN_NEW__
###########################################################################
#
#  Copyright 2024 HPC-Gridware GmbH
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

# Entrypoint script for openSUSE-based OCS container

# Check if OCS is installed, if not, install it
if [ ! -f "${SGE_ROOT}/default/common/settings.sh" ]; then
    echo "OCS not installed. Installing OCS 9.0.7..."
    cd /opt/helpers
    OCS_VERSION="9.0.7" ./ocs.sh --accept-license
    if [ $? -ne 0 ]; then
        echo "Warning: OCS installation failed, continuing with basic setup..."
    fi
fi

# Source OCS environment if available
if [ -f "${SGE_ROOT}/default/common/settings.sh" ]; then
    echo "Sourcing OCS environment..."
    source ${SGE_ROOT}/default/common/settings.sh
else
    echo "OCS environment not found, using default environment variables"
    echo "SGE_ROOT=${SGE_ROOT}"
    echo "SGE_QMASTER_PORT=${SGE_QMASTER_PORT}"
    echo "SGE_EXECD_PORT=${SGE_EXECD_PORT}"
fi

# Check if OCS binaries are available
if command -v qconf >/dev/null 2>&1; then
    echo "OCS commands available:"
    echo "  qconf: $(which qconf)"
    echo "  qsub: $(which qsub)"
    echo "  qstat: $(which qstat)"
else
    echo "Warning: OCS commands not found in PATH"
    echo "Current PATH: $PATH"
fi

# Start OCS services if not already running
echo "Checking OCS services..."
if ! pgrep -f sge_qmaster >/dev/null 2>&1; then
    echo "Starting OCS qmaster..."
    ${SGE_ROOT}/bin/lx-amd64/sge_qmaster &
fi

if ! pgrep -f sge_execd >/dev/null 2>&1; then
    echo "Starting OCS execd..."
    ${SGE_ROOT}/bin/lx-amd64/sge_execd &
fi

# Wait a moment for services to start
sleep 3

# Display cluster status
echo "Cluster status:"
if command -v qhost >/dev/null 2>&1; then
    qhost 2>/dev/null || echo "qhost not available or cluster not ready"
else
    echo "qhost command not available"
fi

# Execute the command passed to the container
if [ $# -gt 0 ]; then
    echo "Executing: $@"
    exec "$@"
else
    echo "Starting interactive shell..."
    exec /bin/bash
fi