#___INFO__MARK_BEGIN_NEW__
###########################################################################
#
#  Copyright 2024-2025 HPC-Gridware GmbH
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
#FROM hpcgridware/clusterscheduler-latest-ubuntu2204:V901_TAG
#FROM hpcgridware/ocs-ubuntu2204:9.0.2
#FROM hpcgridware/ocs-ubuntu2204-nightly:20250203
# standard ubuntu 22.04 - need to install packages
FROM ubuntu:22.04

RUN mkdir -p /opt/helpers

COPY autoinstall.template /opt/helpers/
COPY installer.sh /opt/helpers/
COPY entrypoint.sh /entrypoint.sh

# Download and unpack Cluster Scheduler 9.0.8 tar.gz files for all architectures and components

RUN apt-get update && \
    apt-get install -y wget tar

RUN mkdir -p /opt/ocs

# lx-amd64
RUN wget -O /opt/ocs/ocs-9.0.8-lx-amd64.tar.gz "https://hpc-gridware.com/download/11126/?tmstv=1756559953" && \
    tar -xzf /opt/ocs/ocs-9.0.8-lx-amd64.tar.gz -C /opt/ocs && \
    rm /opt/ocs/ocs-9.0.8-lx-amd64.tar.gz

# lx-arm64
RUN wget -O /opt/ocs/ocs-9.0.8-lx-arm64.tar.gz "https://hpc-gridware.com/download/11128/?tmstv=1756559954" && \
    tar -xzf /opt/ocs/ocs-9.0.8-lx-arm64.tar.gz -C /opt/ocs && \
    rm /opt/ocs/ocs-9.0.8-lx-arm64.tar.gz

# ulx-amd64
RUN wget -O /opt/ocs/ocs-9.0.8-ulx-amd64.tar.gz "https://hpc-gridware.com/download/11132/?tmstv=1756559954" && \
    tar -xzf /opt/ocs/ocs-9.0.8-ulx-amd64.tar.gz -C /opt/ocs && \
    rm /opt/ocs/ocs-9.0.8-ulx-amd64.tar.gz

# doc
RUN wget -O /opt/ocs/ocs-9.0.8-doc.tar.gz "https://hpc-gridware.com/download/11140/?tmstv=1756559954" && \
    tar -xzf /opt/ocs/ocs-9.0.8-doc.tar.gz -C /opt/ocs && \
    rm /opt/ocs/ocs-9.0.8-doc.tar.gz

# common
RUN wget -O /opt/ocs/ocs-9.0.8-common.tar.gz "https://hpc-gridware.com/download/11138/?tmstv=1756559954" && \
    tar -xzf /opt/ocs/ocs-9.0.8-common.tar.gz -C /opt/ocs && \
    rm /opt/ocs/ocs-9.0.8-common.tar.gz

# Install dependencies for Open Cluster Scheduler
RUN apt-get update && apt-get install -y git tar binutils sudo make wget bash libtirpc3 libtirpc-dev

# Install Go
ARG GOLANG_VERSION=1.23.6

RUN apt-get update && \
    apt-get install -y curl wget git gcc make vim libhwloc-dev hwloc software-properties-common man-db  && \
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
