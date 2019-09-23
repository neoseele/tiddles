IMAGE_NAME ?= tiddles
IMAGE_TAG ?= local

.PHONY: build
build: 
	gcloud builds submit --config=cloudbuild.yaml

.PHONY: build-deploy
build-deploy: 
	skaffold run -p gcb

IMAGE := $(shell sh -c "docker image inspect $(IMAGE_NAME):$(IMAGE_TAG) &>/dev/null || echo missing")

.PHONY: build-local
build-local:

ifeq ($(IMAGE),missing)
	@echo "$(IMAGE_NAME):$(IMAGE_TAG) is missing. building ..."
	@docker build -t ${IMAGE_NAME}:$(IMAGE_TAG) .
endif

SERVER := $(shell sh -c "docker inspect $(IMAGE_NAME)-server &>/dev/null || echo missing")

.PHONY: run-server
run-server: build-local

ifeq ($(IMAGE),missing)
	@echo "$(IMAGE_NAME)-server container is not running. starting up ..."
	@docker run -it --rm -d \
		--name $(IMAGE_NAME)-server \
		-p 127.0.0.1:80:80 \
		-p 127.0.0.1:443:443 \
		-p 127.0.0.1:50000:50000 \
		${IMAGE_NAME}:$(IMAGE_TAG) \
		/tiddles --cert=/tls/tls.crt --key=/tls/tls.key
endif

.PHONY: run-client
run-client: run-server

	@docker run -it --rm --network="host" \
		${IMAGE_NAME}:$(IMAGE_TAG) \
		/tiddles --client-only --grpc-backend=localhost:50000 --cert=/tls/tls.crt


