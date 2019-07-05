SHELL = /bin/sh

# Image URL to use all building/pushing image targets
REPO ?= amitkgupta/boshv3-controller

.PHONY: all $(MAKECMDGOALS)

all: _yaml _exe

# Remove the built executable
clean:
	rm -f ./boshv3

# Generate manifests e.g. CRD, RBAC etc.
_yaml: _generator
	$(CONTROLLER_GEN) crd:trivialVersions=true rbac:roleName=manager-role paths="./..." output:crd:artifacts:config=config/crd/bases

# Build local executable
_exe: _code _fmt _vet
	go build

# Generate code
_code: _generator
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...

# Run go fmt against code
_fmt:
	go fmt ./...

# Run go vet against code
_vet:
	go vet ./...

# find or download controller-gen
_generator:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.0-beta.2
CONTROLLER_GEN=$(shell go env GOPATH)/bin/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

# Run against the configured Kubernetes cluster in ~/.kube/config
run: _exe
	kubectl apply -k config/crd
	BOSH_SYSTEM_NAMESPACE=bosh-system ./boshv3

# Build the Docker image
image: _tag
	docker build . -t "${REPO}:${TAG}"

# Push the Docker image to a repo
repo: _tag
	docker push "${REPO}:${TAG}"
	sed -e 's@image: .*@image: '"${REPO}:${TAG}"'@' -i '' ./hack/test/kustomize/manager_deployment_image.yaml
	git commit -am 'h/t/k/manager_deployment_image.yaml: set image to ${REPO}:${TAG}'

# Set the TAG environment variable used for tagging Docker image
_tag:
	test -z "$(git status --porcelain)"
TAG=$(shell git rev-parse --short HEAD)

# Install controller and RBAC in the configured Kubernetes cluster in ~/.kube/config
install:
	kubectl apply -k hack/test/kustomize

# Uninstall controller and RBAC from the configured Kubernetes cluster in ~/.kube/config
uninstall:
	kubectl delete deploy --all -n bosh-system