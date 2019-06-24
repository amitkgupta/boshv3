# Image URL to use all building/pushing image targets
IMG ?= boshv3-controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

.PHONY: all $(MAKECMDGOALS)

all: test

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy-controller: generate-code
	kustomize build config/manager | kubectl apply -f -

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate-code fmt vet
	go run ./main.go

# Install CRDs into a cluster
apply-manifests: generate-base-manifests
	kustomize build config/crd | kubectl apply -f -
	kustomize build config/rbac | kubectl apply -f -

# Push the docker image
docker-push:
	docker push ${IMG}

# Build the docker image
docker-build: test
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Run tests
test: generate-code fmt vet generate-base-manifests
	go test ./api/... ./controllers/... -coverprofile cover.out

# Generate manifests e.g. CRD, RBAC etc.
generate-base-manifests: controller-gen-install
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Generate code
generate-code: controller-gen-install
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# find or download controller-gen
# download controller-gen if necessary
controller-gen-install:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.0-beta.2
CONTROLLER_GEN=$(shell go env GOPATH)/bin/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
