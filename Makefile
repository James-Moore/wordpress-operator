# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?=  controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= jamesjmoore/wordpress-operator:1.0
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests following test environment setup instructions laid out here:
# https://sdk.operatorframework.io/docs/building-operators/golang/references/envtest-setup/
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	wget -nc -q -P ${ENVTEST_ASSETS_DIR} https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/master/hack/setup-envtest.sh
	chmod 755 ${ENVTEST_ASSETS_DIR}/setup-envtest.sh
	bash ${ENVTEST_ASSETS_DIR}/setup-envtest.sh fetch_envtest_tools ${ENVTEST_ASSETS_DIR}
	bash ${ENVTEST_ASSETS_DIR}/setup-envtest.sh setup_envtest_env ${ENVTEST_ASSETS_DIR}
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .


#JAMES MOORE BUILD TARGETS
.PHONY: mydev
mydev: generate fmt vet manifests install
	cd config/default/ && kustomize edit set namespace "default" && cd ../..
	@echo "Dev Make Done"

.PHONY: mybuild
mybuild: mydev docker-build docker-push
	@echo "Build Done"

.PHONY: mystart
mystart: deploy create
	@echo "Start Done"
	#kubectl logs wordpress-operator-controller-manager-7c8ddcb78c-9vfpq manager

.PHONY: mystop
mystop: delete
	@kustomize build config/default | kubectl delete --ignore-not-found -f -

.PHONY: myclean
myclean: mystop
	@dockerlist=$$(docker images -a | grep -E 'none|wordpress' | awk '{print $$3}'); \
	for i in $${dockerlist} ; do  \
		docker rmi -f $${i} ; \
	done

	@ctrlist=$$(microk8s.ctr images list | grep wordpress | awk '{print $$1}'); \
	for i in $${ctrlist} ; do  \
		microk8s.ctr images rm $${i} ; \
	done

	#@docker system prune -af
	@echo "Clean Done"







.PHONY: create
create:  createDeployment createService createIngress
	@echo "Creation Completed"

.PHONY: delete
delete: deleteConfigmap deleteDeployment deleteService deleteIngress
	@echo "Deletion Completed"

.PHONY: createConfigmap
createConfigmap:
	-kubectl create configmap mysqlconfig --from-file ./config/samples/cnf/mysql.cnf
	-kubectl create configmap mysqlsecurecheck --from-file ./config/samples/cnf/blank.cnf

.PHONY: createDeployment
createDeployment:
	-kubectl apply -f ./config/samples/wordpress-fullstack_v1_wordpress.yaml

.PHONY: createService
createService:
	-kubectl apply -f ./config/samples/service/wordpress_service.yaml

.PHONY: createIngress
createIngress:
	-kubectl apply -f ./config/samples/ingress/wordpress_ingress.yaml



.PHONY: deleteConfigmap
deleteConfigmap:
	-kubectl delete configmaps --all

.PHONY: deleteDeployment
deleteDeployment:
	-kubectl delete --ignore-not-found=true -f ./config/samples/wordpress-fullstack_v1_wordpress.yaml

.PHONY: deleteService
deleteService:
	-kubectl delete --ignore-not-found=true -f ./config/samples/service/wordpress_service.yaml

.PHONY: deleteIngress
deleteIngress:
	-kubectl delete --ignore-not-found=true -f ./config/samples/ingress/wordpress_ingress.yaml


.PHONY: describeConfigmap
describeConfigmap:
	-kubectl describe configmaps mysqlconfig
	-kubectl describe configmaps mysqlsecurecheck

.PHONY: describeIngress
describeIngress:
	-kubectl describe ingress wordpress-ingress



.PHONY: shellmysql
shellmysql:
	kubectl exec --stdin --tty $$(kubectl get pods | grep wordpress | awk '{print $$1}') -c mysql -- /bin/bash

.PHONY: shellwordpress
shellwordpress:
	kubectl exec --stdin --tty $$(kubectl get pods | grep wordpress | awk '{print $$1}') -c wordpress -- /bin/bash
