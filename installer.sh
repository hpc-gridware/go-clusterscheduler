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


# This needs to be mounted, the installation directory
# and the host name must be consistent between restarts
export MOUNT_DIR=/opt/cs-install

echo "source ${MOUNT_DIR}/default/common/settings.sh" >> /root/.bashrc

if [ -d ${MOUNT_DIR}/default/common ]; then
  echo "Open Cluster Scheduler seems to be already installed!"
  echo "Starting Open Cluster Scheduler daemons."
  ${MOUNT_DIR}/default/common/sgemaster
  ${MOUNT_DIR}/default/common/sgeexecd
  exit 0
fi

echo "Open Cluster Scheduler is not yet installed in ${MOUNT_DIR}. Starting installation."

# Copy unpacked Open Cluster Scheduler package to ${MOUNT_DIR}
if [ -d /opt/ocs ]; then
  cp -r /opt/ocs/* "${MOUNT_DIR}"
else
  cp -r /opt/cs/* "${MOUNT_DIR}"
fi

cd ${MOUNT_DIR}

# qmon is required for the installer
#mkdir -p /opt/ge-install/bin/lx-amd64
#touch /opt/ge-install/bin/lx-amd64/qmon

cd /opt/helpers
cp autoinstall.template ${MOUNT_DIR}/
cd ${MOUNT_DIR}

# Installer calls: 
#./utilbin/lx-amd64/filestat -owner .
# linux namespaces cause a different ownership of the host mounted
# directory - this causes the installer to abort on Linux

rm ./utilbin/lx-amd64/filestat
echo "#!/bin/bash" > ./utilbin/lx-amd64/filestat
echo "echo root\n" >> ./utilbin/lx-amd64/filestat
chmod +x ./utilbin/lx-amd64/filestat

# install qmaster and execd from scratch when container starts
sed "s:docker:${HOSTNAME}:g" ./autoinstall.template > ./template_host
./inst_sge -m -x -auto ./template_host

# make sure installation is in path and libraries can be accessed
source ${MOUNT_DIR}/default/common/settings.sh
export LD_LIBRARY_PATH=$SGE_ROOT/lib/lx-amd64

# enable that root can submit jobs
qconf -sconf | sed -e 's:100:0:g' > global
qconf -Mconf ./global

# reduce scheduler reaction time to 1 second - and scheduling interval from
# 2 min. to 1 sec.
#qconf -ssconf | sed -e 's:4:1:g' | sed -e 's:2\:0:0\:1:g' > schedconf
#qconf -Msconf ./schedconf

# process 10 jobs at once per node
qconf -rattr queue slots 10 all.q
