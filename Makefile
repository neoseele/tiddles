IMAGE_NAME ?= tiddles
IMAGE_TAG ?= local

.PHONY: build
build:
	gcloud builds submit --config=cloudbuild.yaml

.PHONY: build-deploy
build-deploy:
	skaffold run -p gcb

.PHONY: clean-build-deploy
clean-build-deploy:
	skaffold delete -p gcb

IMAGE := $(shell sh -c "docker image inspect $(IMAGE_NAME):$(IMAGE_TAG) &>/dev/null || echo missing")

.PHONY: build-local
build-local:

ifeq ($(IMAGE),missing)
	@echo "building image [$(IMAGE_NAME):$(IMAGE_TAG)] ..."
	@docker build -t ${IMAGE_NAME}:$(IMAGE_TAG) .
else
	@echo "image [${IMAGE_NAME}:$(IMAGE_TAG)] already exists."
endif

SERVER := $(shell sh -c "docker inspect $(IMAGE_NAME)-server &>/dev/null || echo missing")

.PHONY: run-server
run-server: build-local # depends on build-local

ifeq ($(SERVER),missing)
	@echo "starting up container [$(IMAGE_NAME)-server] ..."
	@docker run -it --rm -d \
		--name $(IMAGE_NAME)-server \
		-p 127.0.0.1:80:80 \
		-p 127.0.0.1:443:443 \
		-p 127.0.0.1:8888:8888 \
		-p 127.0.0.1:50000:50000 \
		${IMAGE_NAME}:$(IMAGE_TAG) \
		/tiddles --cert=/tls/tls.crt --key=/tls/tls.key
else
	@echo "container [$(IMAGE_NAME)-server] is already running."
endif

CLIENT := $(shell sh -c "docker inspect $(IMAGE_NAME)-client &>/dev/null || echo missing")

.PHONY: run-client
run-client: run-server # depends on run-server

ifeq ($(CLIENT),missing)
	@echo "starting up container [$(IMAGE_NAME)-client] ..."
	@docker run -it --rm --network="host" \
		${IMAGE_NAME}:$(IMAGE_TAG) \
		/tiddles --client-only --grpc-backend=localhost:50000 --cert=/tls/tls.crt
else
	@echo "container [$(IMAGE_NAME)-client] is already running."
endif

.PHONY: stop
stop:

ifneq ($(SERVER),missing)
	@echo "stopping container [$(IMAGE_NAME)-server] ..."
	@docker stop $(IMAGE_NAME)-server
endif

ifneq ($(CLIENT),missing)
	@echo "stopping container [$(IMAGE_NAME)-client] ..."
	@docker stop $(IMAGE_NAME)-client
endif

.PHONY: clean
clean:

ifneq ($(IMAGE),missing)
	@echo "removing image [${IMAGE_NAME}:$(IMAGE_TAG)] ..."
	@docker rmi ${IMAGE_NAME}:$(IMAGE_TAG)
endif