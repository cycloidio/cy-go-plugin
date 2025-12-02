VERSION=0.0.2

.PHONY: docker-release
docker-release: ## Builds the base Docker image for the registry
	@docker build -t cycloid/cy-go-plugin:$(VERSION) .
	@docker push cycloid/cy-go-plugin:$(VERSION)
