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

FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    git tar binutils sudo make wget bash curl gcc vim \
    libtirpc3 libtirpc-dev libhwloc-dev hwloc \
    man-db software-properties-common

ARG GOLANG_VERSION=1.24.11
RUN wget -q https://go.dev/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go${GOLANG_VERSION}.linux-amd64.tar.gz && \
    rm go${GOLANG_VERSION}.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin:/root/go/bin

RUN mkdir -p /root/go/bin /root/go/src/github.com/dgruber && \
    cd /root/go/src/github.com/dgruber && \
    git clone https://github.com/dgruber/drmaa.git

RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest

RUN mkdir -p /opt/helpers && \
    wget -q -O /opt/helpers/ocs.sh \
    https://raw.githubusercontent.com/hpc-gridware/quickinstall/refs/heads/main/ocs.sh && \
    chmod +x /opt/helpers/ocs.sh

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN touch /etc/localtime

ENV SGE_ROOT=/opt/ocs
ENV LD_LIBRARY_PATH=${SGE_ROOT}/lib/lx-amd64

WORKDIR /root/go/src/github.com/hpc-gridware/go-clusterscheduler

EXPOSE 6444 6445 7070 8888 9464

ENTRYPOINT ["/entrypoint.sh"]
