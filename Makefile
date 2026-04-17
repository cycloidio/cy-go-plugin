VERSION=0.0.2
LOCAL_REGISTRY=localhost:5000

.PHONY: docker-release
docker-release: ## Builds the base Docker image for the registry
	@docker build -t cycloid/cy-go-plugin:$(VERSION) .
	@docker push cycloid/cy-go-plugin:$(VERSION)

.PHONY: docker-local
docker-local: ## Builds and pushes the Docker image to a local registry
	@docker build --provenance=false -t $(LOCAL_REGISTRY)/cycloid/cy-go-plugin:$(VERSION) .
	@docker push $(LOCAL_REGISTRY)/cycloid/cy-go-plugin:$(VERSION)
