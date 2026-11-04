#!/bin/bash

#___INFO__MARK_BEGIN_NEW__
###########################################################################
#
#  Copyright 2026 HPC-Gridware GmbH
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

# Entrypoint for testing against Gridware Cluster Scheduler (GCS).
#
# GCS packages are not published for download, so unlike entrypoint.sh (which
# fetches Open Cluster Scheduler with the quickinstall script) this installs
# from tarballs that are bind-mounted read-only at ${GCS_PACKAGE_DIR}. The
# installation goes to ${SGE_ROOT}, which the Makefile mounts from the host so
# the cluster survives container restarts.

set -e

SGE_ROOT="/opt/ocs"
GCS_PACKAGE_DIR="${GCS_PACKAGE_DIR:-/packages}"
ARCH="lx-amd64"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ -f "${SGE_ROOT}/default/common/settings.sh" ]; then
    echo "GCS already installed. Starting daemons..."
    source "${SGE_ROOT}/default/common/settings.sh"
    "${SGE_ROOT}/default/common/sgemaster" || true
    "${SGE_ROOT}/default/common/sgeexecd" || true
else
    echo "Installing GCS from ${GCS_PACKAGE_DIR}..."
    for tarball in "${GCS_PACKAGE_DIR}"/gcs-*-common.tar.gz \
                   "${GCS_PACKAGE_DIR}"/gcs-*-bin-${ARCH}.tar.gz \
                   "${GCS_PACKAGE_DIR}"/gcs-*-doc.tar.gz; do
        if [ ! -f "${tarball}" ]; then
            echo "Missing package: ${tarball}" >&2
            exit 1
        fi
        echo "  unpacking $(basename "${tarball}")"
        tar -xzf "${tarball}" -C "${SGE_ROOT}"
    done

    cd "${SGE_ROOT}"

    # The installer runs utilbin/<arch>/filestat -owner . to check that
    # SGE_ROOT belongs to root. A bind-mounted host directory reports a
    # different owner because of the user namespace mapping, which makes the
    # installer abort, so stub the check out.
    printf '#!/bin/bash\necho root\n' > "utilbin/${ARCH}/filestat"
    chmod +x "utilbin/${ARCH}/filestat"

    # The template ships with SGE_ROOT=/opt/cs-install and "docker" as the
    # admin/submit/exec host.
    sed -e "s:/opt/cs-install:${SGE_ROOT}:g" -e "s:docker:$(hostname):g" \
        "${SCRIPT_DIR}/autoinstall.template" > "${SGE_ROOT}/template_host"

    ./inst_sge -m -x -auto "${SGE_ROOT}/template_host"

    source "${SGE_ROOT}/default/common/settings.sh"

    # Tests run as root, which the default min_uid/min_gid of 100 rejects.
    # qconf -Mconf takes the configuration name from the file name, so the
    # file has to be called "global".
    mkdir -p /tmp/gcs-conf
    qconf -sconf | sed -e 's/^\(min_uid\|min_gid\) .*/\1 0/' > /tmp/gcs-conf/global
    qconf -Mconf /tmp/gcs-conf/global

    # Allow 10 single-core jobs at once so job tests do not serialize. The
    # host-specific override keeps all.q shaped like the one the quickinstall
    # setup produces for OCS ("slots 10,[master=14]"), so test results are
    # comparable between the two installation paths.
    qconf -rattr queue slots 10 all.q
    qconf -aattr queue slots "[$(hostname)=10]" all.q
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

export LD_LIBRARY_PATH="${SGE_ROOT}/lib/${ARCH}${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"

cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler 2>/dev/null || true

if [ $# -gt 0 ]; then
    exec "$@"
else
    exec /bin/bash
fi
