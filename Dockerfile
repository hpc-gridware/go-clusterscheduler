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

#FROM hpcgridware/clusterscheduler-latest-ubuntu2204:latest
FROM hpcgridware/clusterscheduler-latest-ubuntu2204:c245a267a

RUN mkdir -p /opt/helpers

COPY autoinstall.template /opt/helpers/
COPY installer.sh /opt/helpers/
COPY entrypoint.sh /entrypoint.sh

ARG GOLANG_VERSION=1.22.4

RUN apt-get update && \
    apt-get install -y wget git gcc make vim libhwloc-dev hwloc software-properties-common && \
    add-apt-repository -y ppa:apptainer/ppa && \
    apt-get update && \
    apt-get install -y apptainer

RUN touch /etc/localtime

RUN wget https://go.dev/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go${GOLANG_VERSION}.linux-amd64.tar.gz && \
    rm go${GOLANG_VERSION}.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin:/root/go/bin

RUN mkdir -p /root/go/bin && \
    mkdir -p /root/go/src/github.com/dgruber

RUN cd /root/go/src/github.com/dgruber && \
    git clone https://github.com/dgruber/drmaa.git

RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest

WORKDIR /root/go/src/github.com/dgruber

ENV SGE_ROOT=/opt/cs-install
ENV LD_LIBRARY_PATH=${SGE_ROOT}/lib/lx-amd64

ENTRYPOINT [ "/entrypoint.sh" ]
