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

# OCS 9.1.7 and later install with MUNGE authentication by default. The
# installation in ${SGE_ROOT} is persisted, but the MUNGE key and munged
# live in the container and are gone after a restart, so the qmaster would
# not start. A new key is fine on this single-host cluster: MUNGE
# credentials are short-lived and never stored.
start_munge() {
    if [ ! -f /etc/munge/munge.key ]; then
        echo "Creating MUNGE key..."
        (umask 077 && dd if=/dev/urandom bs=1 count=1024 \
            of=/etc/munge/munge.key 2>/dev/null)
        chown munge:munge /etc/munge/munge.key
        chmod 0400 /etc/munge/munge.key
    fi
    # No systemd in the container, so neither tmpfiles nor a unit
    # prepares /run/munge and starts munged.
    mkdir -p /run/munge
    chown munge:munge /run/munge
    chmod 755 /run/munge
    runuser -u munge -- /usr/sbin/munged
}

if [ -f "${SGE_ROOT}/default/common/settings.sh" ]; then
    echo "OCS already installed. Starting daemons..."
    source "${SGE_ROOT}/default/common/settings.sh"
    if grep -q '^security_mode.*munge' "${SGE_ROOT}/default/common/bootstrap"; then
        start_munge
    fi
    "${SGE_ROOT}/default/common/sgemaster" || true
    "${SGE_ROOT}/default/common/sgeexecd" || true
else
    echo "Installing OCS ${OCS_VERSION:-9.1.5}..."
    cd /tmp
    OCS_VERSION="${OCS_VERSION:-9.1.5}" /opt/helpers/ocs.sh
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
