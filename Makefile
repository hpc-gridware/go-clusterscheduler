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

# This Makefile is used for development and testing purposes only.

IMAGE_NAME = $(shell basename $(CURDIR))
IMAGE_TAG = V901_TAG
CONTAINER_NAME = $(IMAGE_NAME)

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
	docker run -p 7070:7070 --rm -it -h master --privileged -v /dev/fuse:/dev/fuse --cap-add SYS_ADMIN --name $(CONTAINER_NAME) -v ./installation:/opt/cs-install -v ./:/root/go/src/github.com/hpc-gridware/go-clusterscheduler $(IMAGE_NAME):$(IMAGE_TAG) /bin/bash

.PHONY: run
run: build
	@echo "Running the Open Cluster Scheduler container..."
	@echo "For a new installation, you need to remove the ./installation subdirectory first."
	mkdir -p ./installation
	docker run --platform=linux/amd64 -p 7070:7070 -p 9464:9464 --rm -it -h master --name $(CONTAINER_NAME) -v ./installation:/opt/cs-install -v ./:/root/go/src/github.com/hpc-gridware/go-clusterscheduler $(IMAGE_NAME):$(IMAGE_TAG) /bin/bash

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
	docker run --rm -it -h master --privileged --cap-add SYS_ADMIN -p 9464:9464 --name $(CONTAINER_NAME) -v ./installation:/opt/cs-install -v ./:/root/go/src/github.com/hpc-gridware/go-clusterscheduler $(IMAGE_NAME):$(IMAGE_TAG) /bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler/cmd/simulator && go build . && ./simulator run ../../cluster.json && /bin/bash"

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
	docker run --rm -it -h master -p 8282:8282 --name $(CONTAINER_NAME) -v ./installation:/opt/cs-install -v ./:/root/go/src/github.com/hpc-gridware/go-clusterscheduler $(IMAGE_NAME):$(IMAGE_TAG) /bin/bash -c "cd /root/go/src/github.com/hpc-gridware/go-clusterscheduler/cmd/adapter && go build . && ./adapter"

.PHONY: clean
clean:
	@echo "Removing the container..."
	docker rm -f $(CONTAINER_NAME) || true
	@echo "Removing the image..."
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) || true
	@echo "Removing the installation directory..."
	rm -rf ./installation
