IMAGE_NAME ?= tiddles
IMAGE_TAG ?= latest
REPO_NAME ?= neoseele

.PHONY: build
build:
	gcloud builds submit --config=cloudbuild.yaml

.PHONY: build-deploy
build-deploy:
	skaffold run -p gcb

.PHONY: clean-build-deploy
clean-build-deploy:
	skaffold delete -p gcb

IMAGE := $(shell docker image inspect $(IMAGE_NAME):$(IMAGE_TAG) &>/dev/null || echo missing)

.PHONY: build-local
build-local:

ifeq ($(IMAGE),missing)
	@echo "building image [$(IMAGE_NAME):$(IMAGE_TAG)] ..."
	@docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
else
	@echo "image [$(IMAGE_NAME):$(IMAGE_TAG)] already exists."
endif

.PHONY: build-dockerhub
build-dockerhub: build-local # depend on build-local
	@image_id=$$(docker images $(IMAGE_NAME):$(IMAGE_TAG) --format '{{.ID}}') && \
	if [ -n "$$image_id" ]; then \
		echo "$$image_id"; \
		docker tag $$image_id $(REPO_NAME)/$(IMAGE_NAME):$(IMAGE_TAG); \
		docker push $(REPO_NAME)/$(IMAGE_NAME):$(IMAGE_TAG); \
	fi

.PHONY: run-server
run-server: build-local # depends on build-local
	@echo "starting up container [$(IMAGE_NAME)-server] ..."
	@if docker inspect $(IMAGE_NAME)-server &>/dev/null; then \
		echo "container [$(IMAGE_NAME)-server] is already running."; \
	else \
		docker run -it --rm -d \
			--name $(IMAGE_NAME)-server \
			-p 127.0.0.1:80:80 \
			-p 127.0.0.1:443:443 \
			-p 127.0.0.1:8888:8888 \
			-p 127.0.0.1:50000:50000 \
			$(IMAGE_NAME):$(IMAGE_TAG) \
			/tiddles --cert=/tls/tls.crt --key=/tls/tls.key; \
	fi

.PHONY: run-client
run-client: run-server # depends on run-server
	@echo "starting up container [$(IMAGE_NAME)-client] ..."
	@if docker inspect $(IMAGE_NAME)-client &>/dev/null; then \
		echo "container [$(IMAGE_NAME)-client] is already running."; \
	else \
		docker run -it --rm --network="host" \
			$(IMAGE_NAME):$(IMAGE_TAG) \
			/tiddles --client-only --grpc-backend=localhost:50000 --cert=/tls/tls.crt; \
	fi

.PHONY: stop
stop:
	@echo "stopping container [$(IMAGE_NAME)-server] ..."
	-@docker stop $(IMAGE_NAME)-server

.PHONY: clean
clean: stop # stop the server if it is still running
	@echo "removing images ..."
	-@docker rmi $(IMAGE_NAME):$(IMAGE_TAG)
	-@docker rmi $(REPO_NAME)/$(IMAGE_NAME):$(IMAGE_TAG)
