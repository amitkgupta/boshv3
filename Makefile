# Namespace where all system-level resource for BOSH v3 will be installed
BOSH_SYSTEM_NAMESPACE ?= bosh-system
# Image URL to use all building/pushing image targets
IMG ?= amitkgupta/boshv3-controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

.PHONY: exe $(MAKECMDGOALS)

# Build local executable
exe: code fmt vet
	go build

# Generate manifests e.g. CRD, RBAC etc.
yaml: generator
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Install CRDs into a cluster
crd: yaml
	kustomize build config/crd | kubectl apply -f -

# Run against the configured Kubernetes cluster in ~/.kube/config
run: exe crd
	BOSH_SYSTEM_NAMESPACE="${BOSH_SYSTEM_NAMESPACE}" ./boshv3

# Build the docker image
image: exe
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -e 's@image: .*@image: '"${IMG}"'@' -i '' ./config/manager/patches/manager_image_patch.yaml

# Push the docker image to a repo
repo:
	docker push ${IMG}

# Install controller and RBAC in the configured Kubernetes cluster in ~/.kube/config
install: crd
	kustomize build config/rbac | kubectl apply -f -
	kustomize build config/manager | kubectl apply -f -

# Generate code
code: generator
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# find or download controller-gen
generator:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.0-beta.2
CONTROLLER_GEN=$(shell go env GOPATH)/bin/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
