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

# --------------------------------------------------------------------
# This configuration is intended for local development on Darwin/macOS 
# with ARM architecture (such as M4). 
# 
# Usage on Linux systems and/or AMD64 architectures may require 
# adjustments to ensure images build and run correctly. 
# 
# Contributions to broaden compatibility are welcome!
# --------------------------------------------------------------------

IMAGE_NAME = $(shell basename $(CURDIR))
IMAGE_TAG = V902_TAG
CONTAINER_NAME = $(IMAGE_NAME)

# openSUSE-based images for testing against released versions
OPENSUSE_IMAGE_NAME = $(IMAGE_NAME)-opensuse
OPENSUSE_IMAGE_TAG = leap15.6-ocs907
OPENSUSE_CONTAINER_NAME = $(OPENSUSE_IMAGE_NAME)

.PHONY: build
build:
	@echo "Building the Open Cluster Scheduler image..."
	docker build --platform=linux/amd64 -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Running apptainers in containers requires more permissions. You can drop
# the --privileged flag and the --cap-add SYS_ADMIN flag if you don't need
# to run apptainers in containers.
.PHONY: run-privileged
run-privileged: build
	@echo "Running the Open Cluster Scheduler container in privileged mode..."
	mkdir -p ./installation
	docker run -p 7070:7070 --rm -it -h master \
		--privileged -v /dev/fuse:/dev/fuse --cap-add SYS_ADMIN \
		--name $(CONTAINER_NAME) \
		-v ${PWD}/installation:/opt/cs-install \
		-v ${PWD}/:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(IMAGE_NAME):$(IMAGE_TAG) /bin/bash

.PHONY: run
run: build
	@echo "Running the Open Cluster Scheduler container..."
	@echo "For a new installation, you need to remove the ./installation subdirectory first."
	mkdir -p ./installation
	docker run --platform=linux/amd64 --rm -it -h master \
		-p 8889:8888 -p 7070:7070 \
		--name $(CONTAINER_NAME) \
		-v ${PWD}/installation:/opt/cs-install \
		-v ${PWD}/:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(IMAGE_NAME):$(IMAGE_TAG) /bin/bash

# Running apptainers in containers requires more permissions. You can drop
# the --privileged flag and the --cap-add SYS_ADMIN flag if you don't need
# to run apptainers in containers.
.PHONY: simulate
simulate:
	@echo "Running the container in simulation mode using cluster.json"
	@echo "Removing subdirectory with old installation..."
	rm -rf ./installation
	@echo "Creating new subdirectory for installation..."
	mkdir -p ./installation
	docker run --platform=linux/amd64 --rm -it -h master \
		--privileged --cap-add SYS_ADMIN \
		-p 8080:8080 -p 9464:9464 -p 8888:8888 \
		--name $(CONTAINER_NAME) \
		-v ${PWD}/installation:/opt/cs-install \
		-v ${PWD}/:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(IMAGE_NAME):$(IMAGE_TAG) \
		/bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler/cmd/simulator && \
		go mod tidy && \
		GOFLAGS=-buildvcs=false go build . && \
		./simulator run ../../cluster.json && \
		/bin/bash"


#.PHONY: simulate
#simulate:
#	@echo "Running the container in simulation mode using cluster.json"
#	mkdir -p ./installation
#	docker run --rm -it -h master --name $(CONTAINER_NAME) -v ./installation:/opt/cs-install -v ./:/root/go/src/github.com/hpc-gridware/go-clusterscheduler $(IMAGE_NAME):$(IMAGE_TAG) /bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler/cmd/simulator && go build . && ./simulator run ../../cluster.json && /bin/bash"

.PHONY: adapter
adapter:
	@echo "Running the adapter on port 8282...POST to http://localhost:8282/api/v0/command"
	@echo "Example: curl -X POST http://localhost:8282/api/v0/command -d '{\"method\": \"ShowSchedulerConfiguration\"}'"
	mkdir -p ./installation
	docker run --platform=linux/amd64 --rm -it -h master \
		-p 8282:8282 \
		--name $(CONTAINER_NAME) \
		-v ${PWD}/installation:/opt/cs-install \
		-v ${PWD}/:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(IMAGE_NAME):$(IMAGE_TAG) \
		/bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler/cmd/adapter && \
		GOFLAGS=-buildvcs=false go build . && \
		./adapter"

.PHONY: run-rest
run-rest: build
	@echo "Running the Open Cluster Scheduler container with REST adapter..."
	@echo "For a new installation, you need to remove the ./installation subdirectory first."
	mkdir -p ./installation
	docker run --platform=linux/amd64 --rm -it -h master \
		-p 7070:7070 -p 9464:9464 -p 9898:9898 \
		--name $(CONTAINER_NAME) \
		-v ${PWD}/installation:/opt/cs-install \
		-v ${PWD}/:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(IMAGE_NAME):$(IMAGE_TAG) \
		/bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler/cmd/adapter && \
		GOFLAGS=-buildvcs=false go build . && \
		./adapter --port 9898 & exec bash"

.PHONY: test
test: build
	@echo "Running unit tests in container (no cluster required)..."
	docker run --platform=linux/amd64 --rm \
		-v ${PWD}:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(IMAGE_NAME):$(IMAGE_TAG) \
		/bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler && go test ./pkg/helper/... ./pkg/accounting/... ./pkg/adapter/... -v"

.PHONY: test-all
test-all: build
	@echo "Running all tests in container (includes expected failures for integration tests)..."
	docker run --platform=linux/amd64 --rm \
		-v ${PWD}:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(IMAGE_NAME):$(IMAGE_TAG) \
		/bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler && go test ./pkg/... -v"

.PHONY: test-integration
test-integration: build
	@echo "Running integration tests in container with cluster..."
	@echo "This requires interactive container with cluster setup. Use 'make run' for full integration testing."
	mkdir -p ./installation
	docker run --platform=linux/amd64 --rm -it -h master \
		-v ${PWD}/installation:/opt/cs-install \
		-v ${PWD}/:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(IMAGE_NAME):$(IMAGE_TAG) \
		/bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler && echo 'Run: find ./pkg -name \"*_test.go\" -path \"*/v9.0/*\" | xargs -I {} dirname {} | sort -u | xargs -I {} sh -c \"cd {} && ginkgo -v\"' && /bin/bash"

# openSUSE-based targets for testing against released versions
.PHONY: build-opensuse
build-opensuse:
	@echo "Building openSUSE-based OCS image for released version testing..."
	docker build --platform=linux/amd64 -f Dockerfile.opensuse -t $(OPENSUSE_IMAGE_NAME):$(OPENSUSE_IMAGE_TAG) .

.PHONY: test-opensuse
test-opensuse: build-opensuse
	@echo "Running unit tests in openSUSE container against released OCS version..."
	docker run --platform=linux/amd64 --rm \
		-v ${PWD}:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(OPENSUSE_IMAGE_NAME):$(OPENSUSE_IMAGE_TAG) \
		/bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler && go test ./pkg/helper/... ./pkg/accounting/... ./pkg/adapter/... -v"

.PHONY: test-opensuse-all
test-opensuse-all: build-opensuse
	@echo "Running all tests in openSUSE container against released OCS version..."
	docker run --platform=linux/amd64 --rm \
		-v ${PWD}:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(OPENSUSE_IMAGE_NAME):$(OPENSUSE_IMAGE_TAG) \
		/bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler && go test ./pkg/... -v"

.PHONY: run-opensuse
run-opensuse: build-opensuse
	@echo "Running openSUSE-based OCS container for interactive testing..."
	docker run --platform=linux/amd64 -p 6444:6444 -p 6445:6445 -p 8888:8888 --rm -it -h master \
		--name $(OPENSUSE_CONTAINER_NAME) \
		-v ${PWD}:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(OPENSUSE_IMAGE_NAME):$(OPENSUSE_IMAGE_TAG) /bin/bash

.PHONY: test-integration-opensuse
test-integration-opensuse: build-opensuse
	@echo "Running integration tests in openSUSE container with released OCS..."
	@echo "This provides an interactive environment for testing against released versions."
	docker run --platform=linux/amd64 --rm -it -h master \
		-v ${PWD}:/root/go/src/github.com/hpc-gridware/go-clusterscheduler \
		$(OPENSUSE_IMAGE_NAME):$(OPENSUSE_IMAGE_TAG) \
		/bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler && echo 'Testing against released OCS version. Run: find ./pkg -name \"*_test.go\" -path \"*/v9.0/*\" | xargs -I {} dirname {} | sort -u | xargs -I {} sh -c \"cd {} && ginkgo -v\"' && /bin/bash"

.PHONY: clean
clean:
	@echo "Removing the container..."
	docker rm -f $(CONTAINER_NAME) || true
	@echo "Removing the image..."
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) || true
	@echo "Removing the installation directory..."
	rm -rf ./installation

.PHONY: clean-opensuse
clean-opensuse:
	@echo "Removing openSUSE containers and images..."
	docker rm -f $(OPENSUSE_CONTAINER_NAME) || true
	docker rmi $(OPENSUSE_IMAGE_NAME):$(OPENSUSE_IMAGE_TAG) || true

.PHONY: clean-all
clean-all: clean clean-opensuse
	@echo "All containers and images removed."
