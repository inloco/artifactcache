IMAGE_NAME ?= inloco/artifactcache
IMAGE_VERSION ?= $(shell git describe --dirty --broken --always)

# Build, tag, and push the docker image
docker: docker-build docker-tag docker-push

# Build the docker image
docker-build:
	docker build --tag $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT) .

# Tag the docker image
docker-tag:
	docker tag $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT) $(IMAGE_NAME)

# Push the docker image
docker-push:
	docker push $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)
	docker push $(IMAGE_NAME)
