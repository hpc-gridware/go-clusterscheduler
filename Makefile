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

# OCS version to install inside the container (override: make run OCS_VERSION=9.1.4)
# Supported: 9.0.5 - 9.0.12 and 9.1.0 - 9.1.5; the authoritative list is the
# quickinstall installer (ocs.sh), which is re-fetched on every image build so
# new releases become available without touching this repo.
# Note: ./installation persists an installed cluster. Remove it when switching
# OCS_VERSION, otherwise the previously installed version is booted again.
OCS_VERSION    ?= 9.1.5

IMAGE_NAME      = go-clusterscheduler
CONTAINER_NAME  = $(IMAGE_NAME)
PLATFORM        = linux/amd64
PROJECT_DIR     = /root/go/src/github.com/hpc-gridware/go-clusterscheduler

# Gridware Cluster Scheduler (GCS) is not downloadable, so the run-gcs and
# test-gcs targets install it from tarballs kept in ./packages. Point
# GCS_PACKAGES at another directory to test a different GCS release; the
# matching installation directory keeps the versions apart.
# (override: make run-gcs GCS_PACKAGES=packages/gcs910)
GCS_PACKAGES    ?= packages/gcs914
GCS_INSTALL_DIR  = installation-$(notdir $(GCS_PACKAGES))

GCS_RUN_FLAGS = --platform=$(PLATFORM) --rm -h master \
	--entrypoint $(PROJECT_DIR)/entrypoint-gcs.sh \
	-v $(CURDIR)/$(GCS_INSTALL_DIR):/opt/ocs \
	-v $(CURDIR)/$(GCS_PACKAGES):/packages:ro \
	-v $(CURDIR):$(PROJECT_DIR)

.PHONY: build run run-privileged test test-all test-integration simulate adapter run-rest \
	run-gcs test-gcs clean

build:
	docker build --platform=$(PLATFORM) \
		--build-arg OCS_INSTALLER_REFRESH=$(shell date +%s) \
		-t $(IMAGE_NAME) .

run: build
	mkdir -p ./installation
	docker run --platform=$(PLATFORM) --rm -it -h master \
		-e OCS_VERSION=$(OCS_VERSION) \
		-p 7070:7070 -p 8888:8888 \
		--name $(CONTAINER_NAME) \
		-v $(CURDIR)/installation:/opt/ocs \
		-v $(CURDIR):$(PROJECT_DIR) \
		$(IMAGE_NAME)

run-privileged: build
	mkdir -p ./installation
	docker run --platform=$(PLATFORM) --rm -it -h master \
		--privileged -v /dev/fuse:/dev/fuse --cap-add SYS_ADMIN \
		-e OCS_VERSION=$(OCS_VERSION) \
		-p 7070:7070 -p 8888:8888 \
		--name $(CONTAINER_NAME) \
		-v $(CURDIR)/installation:/opt/ocs \
		-v $(CURDIR):$(PROJECT_DIR) \
		$(IMAGE_NAME)

test: build
	docker run --platform=$(PLATFORM) --rm \
		--entrypoint /bin/bash \
		-v $(CURDIR):$(PROJECT_DIR) \
		$(IMAGE_NAME) \
		-c "cd $(PROJECT_DIR) && go test ./pkg/helper/... ./pkg/accounting/... ./pkg/adapter/... -v"

test-all: build
	docker run --platform=$(PLATFORM) --rm \
		--entrypoint /bin/bash \
		-v $(CURDIR):$(PROJECT_DIR) \
		$(IMAGE_NAME) \
		-c "cd $(PROJECT_DIR) && go test ./pkg/... -v"

test-integration: build
	mkdir -p ./installation
	docker run --platform=$(PLATFORM) --rm -it -h master \
		-e OCS_VERSION=$(OCS_VERSION) \
		-v $(CURDIR)/installation:/opt/ocs \
		-v $(CURDIR):$(PROJECT_DIR) \
		$(IMAGE_NAME)

run-gcs: build
	mkdir -p ./$(GCS_INSTALL_DIR)
	docker run $(GCS_RUN_FLAGS) -it \
		-p 7070:7070 -p 8888:8888 \
		--name $(CONTAINER_NAME) \
		$(IMAGE_NAME)

# -p 1 because the cluster-backed packages share one cluster and interfere
# when go test runs them concurrently.
test-gcs: build
	rm -rf ./$(GCS_INSTALL_DIR) && mkdir -p ./$(GCS_INSTALL_DIR)
	docker run $(GCS_RUN_FLAGS) \
		$(IMAGE_NAME) \
		/bin/bash -c "cd $(PROJECT_DIR) && go test -p 1 ./pkg/... -v"

simulate: build
	rm -rf ./installation && mkdir -p ./installation
	docker run --platform=$(PLATFORM) --rm -it -h master \
		--privileged --cap-add SYS_ADMIN \
		-e OCS_VERSION=$(OCS_VERSION) \
		-p 8080:8080 -p 9464:9464 -p 8888:8888 \
		--name $(CONTAINER_NAME) \
		-v $(CURDIR)/installation:/opt/ocs \
		-v $(CURDIR):$(PROJECT_DIR) \
		$(IMAGE_NAME) \
		/bin/bash -c "cd $(PROJECT_DIR)/cmd/simulator && \
			GOFLAGS=-buildvcs=false go build . && \
			sleep 30 && \
			./simulator run ../../cluster.json && \
			/bin/bash"

adapter: build
	mkdir -p ./installation
	docker run --platform=$(PLATFORM) --rm -it -h master \
		-e OCS_VERSION=$(OCS_VERSION) \
		-p 8282:8282 \
		--name $(CONTAINER_NAME) \
		-v $(CURDIR)/installation:/opt/ocs \
		-v $(CURDIR):$(PROJECT_DIR) \
		$(IMAGE_NAME) \
		/bin/bash -c "cd $(PROJECT_DIR)/cmd/adapter && \
			GOFLAGS=-buildvcs=false go build . && \
			./adapter"

run-rest: build
	mkdir -p ./installation
	docker run --platform=$(PLATFORM) --rm -it -h master \
		-e OCS_VERSION=$(OCS_VERSION) \
		-p 7070:7070 -p 9464:9464 -p 9898:9898 \
		--name $(CONTAINER_NAME) \
		-v $(CURDIR)/installation:/opt/ocs \
		-v $(CURDIR):$(PROJECT_DIR) \
		$(IMAGE_NAME) \
		/bin/bash -c "cd $(PROJECT_DIR)/cmd/adapter && \
			GOFLAGS=-buildvcs=false go build . && \
			./adapter --port 9898 & exec bash"

clean:
	docker rm -f $(CONTAINER_NAME) || true
	docker rmi $(IMAGE_NAME) || true
	rm -rf ./installation ./installation-gcs*
