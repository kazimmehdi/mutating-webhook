.PHONY: build docker-build docker-push helm-install helm-uninstall test clean

IMAGE_NAME ?= k8s-mutating-webhook
IMAGE_TAG ?= 1.0.2
REGISTRY ?= artifactory.cloud.cms.gov/cms-devops-docker-local
NAMESPACE ?= webhook-system
RELEASE_NAME ?= mutating-webhook

build:
	go build -o webhook main.go

test:
	go test -v ./...

docker-build:
	docker build -t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) .

docker-build-minikube:
	eval $$(minikube docker-env) && docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

docker-push:
	docker tag $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

helm-install:
	helm upgrade --install ${RELEASE_NAME} ./helm-chart -n $(NAMESPACE) --create-namespace --debug

helm-upgrade:
	helm upgrade ${RELEASE_NAME} ./helm-chart -n $(NAMESPACE) --debug

helm-uninstall:
	helm uninstall ${RELEASE_NAME} -n $(NAMESPACE)

show-logs:
	kubectl logs -l app.kubernetes.io/name=${RELEASE_NAME} -n $(NAMESPACE)

test-webhook-without-topology:
	kubectl apply -f ./examples/test-pod-without-topology.yaml 
	sleep 2
	kubectl get pod test-pod --show-labels
	sleep 2
	kubectl get pod test-pod -o yaml
	sleep 2
	kubectl delete pod test-pod

test-webhook-with-topology:
	kubectl apply -f ./examples/test-pod-with-topology.yaml 
	sleep 2
	kubectl get pod test-pod --show-labels
	sleep 2
	kubectl get pod test-pod -o yaml
	sleep 2
	kubectl delete pod test-pod

test-webhook-with-dupe-topology:
	kubectl apply -f ./examples/test-pod-with-topology.yaml 
	sleep 2
	kubectl get pod test-pod --show-labels
	sleep 2
	kubectl get pod test-pod -o yaml
	sleep 2
	kubectl delete pod test-pod	

clean:
	rm -f webhook
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) || true

help:
	@echo "Available targets:"
	@echo "  build                - Build the Go binary"
	@echo "  docker-build         - Build Docker image"
	@echo "  docker-build-minikube- Build Docker image for Minikube"
	@echo "  docker-push          - Push Docker image to registry"
	@echo "  helm-install         - Install Helm chart"
	@echo "  helm-upgrade         - Upgrade Helm chart"
	@echo "  helm-uninstall       - Uninstall Helm chart"
	@echo "  test-webhook         - Test the webhook"
	@echo "  clean                - Clean build artifacts"
