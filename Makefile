.SUFFIXES:

# Default values if not already set
ANSIBLE_VERSION ?= 2.9.*
PGOROOT ?= $(CURDIR)
PGO_BASEOS ?= centos7
PGO_IMAGE_PREFIX ?= crunchydata
PGO_IMAGE_TAG ?= $(PGO_BASEOS)-$(PGO_VERSION)
PGO_VERSION ?= 4.5.0
PGO_PG_VERSION ?= 12
PGO_PG_FULLVERSION ?= 12.4
PGO_BACKREST_VERSION ?= 2.29
PACKAGER ?= yum

RELTMPDIR=/tmp/release.$(PGO_VERSION)
RELFILE=/tmp/postgres-operator.$(PGO_VERSION).tar.gz

ifneq ($(IMGBUILDER),)
$(warning WARNING: IMGBUILDER is deprecated; images are always built with buildah)
ifeq ($(IMGBUILDER),docker)
$(warning WARNING: set CONTAINER=docker to build images by running docker containers)
CONTAINER ?= docker
endif
endif

# The utility to use when pushing/pulling to and from an image repo (e.g. docker or buildah)
IMG_PUSHER_PULLER ?= docker

ifneq ($(IMG_PUSH_TO_DOCKER_DAEMON),)
$(warning WARNING: IMG_PUSH_TO_DOCKER_DAEMON is deprecated)
ifeq ("$(IMG_PUSH_TO_DOCKER_DAEMON)", "true")
$(warning WARNING: set BUILDAH_TRANSPORT=docker-daemon: to push images as they are built)
BUILDAH_TRANSPORT ?= docker-daemon:
endif
endif

ifneq ($(IMG_ROOTLESS_BUILD),)
$(warning WARNING: IMG_ROOTLESS_BUILD is deprecated; images are rootless by default)
ifneq ("$(IMG_ROOTLESS_BUILD)", "true")
$(warning WARNING: set BUILDAH_SUDO=sudo to build images as root)
BUILDAH_SUDO ?= sudo
endif
endif


DFSET=$(PGO_BASEOS)
DOCKERBASEREGISTRY=registry.access.redhat.com/


# Allows consolidation of ubi/rhel/centos Dockerfile sets
ifeq ("$(PGO_BASEOS)", "rhel7")
        DFSET=rhel
endif

ifeq ("$(PGO_BASEOS)", "ubi7")
        DFSET=rhel
endif

ifeq ("$(PGO_BASEOS)", "ubi8")
        DFSET=rhel
        PACKAGER=dnf
endif

ifeq ("$(PGO_BASEOS)", "centos7")
        DFSET=centos
        DOCKERBASEREGISTRY=centos:
endif

ifeq ("$(PGO_BASEOS)", "centos8")
        DFSET=centos
        PACKAGER=dnf
        DOCKERBASEREGISTRY=centos:
endif

BUILDAH_BUILD_ARGS ?= ## Extra arguments passed to `buildah build-using-dockerfile`.
BUILDAH_CMD = $(BUILDAH_SUDO) buildah
BUILDAH_IMAGE ?= quay.io/buildah/stable:latest
BUILDAH_SUDO ?= ## Set to "sudo" to disable rootless image builds.
BUILDAH_TRANSPORT ?= ## Set to a Buildah transport to push images as they are built. See buildah-push(1).

CONTAINER ?= ## Executable used to build in containers, e.g. "podman".

GO ?= go
GO_BUILD = $(GO_CMD) build
GO_CMD = $(GO_ENV) $(GO)
GO_IMAGE ?= registry.access.redhat.com/ubi8/go-toolset:latest

# Disable optimizations if creating a debug build
ifeq ("$(DEBUG_BUILD)", "true")
	GO_BUILD += -gcflags='all=-N -l'
endif

ifeq (docker,$(findstring docker,$(CONTAINER)))
BUILDAH_CONTAINER_ARGS += --volume '/var/run/docker.sock:/var/run/docker.sock'
$(eval BUILDAH_TRANSPORT = $(or $(value BUILDAH_TRANSPORT),docker-daemon:))
endif

ifeq (podman,$(findstring podman,$(CONTAINER)))
# Run containerized Buildah with the same storage driver as this user.
BUILDAH_ARGS += $(if $(BUILDAH_SUDO),,--storage-driver $(shell buildah info --format '{{.store.GraphDriverName}}'))
BUILDAH_CONTAINER_ARGS += --volume '$(BUILDAH_STORAGE):/var/lib/containers/storage'
BUILDAH_STORAGE ?= $(shell $(BUILDAH_SUDO) buildah info --format '{{.store.GraphRoot}}')
GO_CONTAINER_ARGS += --userns 'keep-id'
endif

ifneq ($(CONTAINER),)
BUILDAH_CMD = $(BUILDAH_SUDO) $(CONTAINER) run --rm \
	--device '/dev/fuse:rw' --network 'host' --security-opt 'seccomp=unconfined' \
	--volume '$(CURDIR):/mnt:delegated' --workdir '/mnt' \
	$(BUILDAH_CONTAINER_ARGS) $(BUILDAH_IMAGE) buildah $(BUILDAH_ARGS)

GO_CMD = $(CONTAINER) run --rm --user "$$(id -u)" \
	--env 'GOCACHE=/tmp/go-build' --env 'GOPATH=/tmp/go-path' \
	--volume '$(CURDIR):/mnt:delegated' --workdir '/mnt' \
	$(GO_CONTAINER_ARGS) $(GO_IMAGE) env $(GO_ENV) go
endif # $(CONTAINER)


.PHONY: all installrbac setup setupnamespaces cleannamespaces \
	deployoperator cli-docs clean push pull release


#======= Main functions =======
all: linuxpgo all-images

installrbac:
	PGOROOT='$(PGOROOT)' ./deploy/install-rbac.sh

setup:
	PGOROOT='$(PGOROOT)' ./bin/get-deps.sh
	./bin/check-deps.sh

setupnamespaces:
	PGOROOT='$(PGOROOT)' ./deploy/setupnamespaces.sh

cleannamespaces:
	PGOROOT='$(PGOROOT)' ./deploy/cleannamespaces.sh

deployoperator:
	PGOROOT='$(PGOROOT)' ./deploy/deploy.sh


#======= Binary builds =======
build-pgo-apiserver:
	$(GO_BUILD) -o bin/apiserver ./cmd/apiserver

build-pgo-backrest:
	$(GO_BUILD) -o bin/pgo-backrest/pgo-backrest ./cmd/pgo-backrest

build-pgo-rmdata:
	$(GO_BUILD) -o bin/pgo-rmdata/pgo-rmdata ./cmd/pgo-rmdata

build-pgo-scheduler:
	$(GO_BUILD) -o bin/pgo-scheduler/pgo-scheduler ./cmd/pgo-scheduler

build-postgres-operator:
	$(GO_BUILD) -o bin/postgres-operator ./cmd/postgres-operator

build-pgo-client:
	$(GO_BUILD) -o bin/pgo ./cmd/pgo

linuxpgo: GO_ENV += GOOS=linux GOARCH=amd64
linuxpgo:
	$(GO_BUILD) -o bin/pgo ./cmd/pgo

macpgo: GO_ENV += GOOS=darwin GOARCH=amd64
macpgo:
	$(GO_BUILD) -o bin/pgo-mac ./cmd/pgo

winpgo: GO_ENV += GOOS=windows GOARCH=386
winpgo:
	$(GO_BUILD) -o bin/pgo.exe ./cmd/pgo


#======= Image builds =======
DOCKERFILES = $(wildcard build/*/Dockerfile)
IMAGES = $(DOCKERFILES:build/%/Dockerfile=image-%)
OTHER_IMAGES = $(filter-out image-pgo-base,$(IMAGES))

ifndef IMG_PUSH_TO_DOCKER_DAEMON
ifeq ($(BUILDAH_TRANSPORT),)
$(IMAGES): --docker-push-notification
--docker-push-notification:
	$(warning INFO: images are no longer pushed to the local Docker daemon by default)
	$(warning INFO: set CONTAINER=docker or BUILDAH_TRANSPORT=docker-daemon: to push images as they are built)
endif
endif

ifndef IMG_ROOTLESS_BUILD
ifeq ($(BUILDAH_SUDO),)
$(IMAGES): --rootless-notification
--rootless-notification:
	$(warning INFO: images are now built by the current user rather than root)
	$(warning INFO: set BUILDAH_SUDO=sudo to build images as root)
endif
endif

.PHONY: all-images $(IMAGES) $(IMAGES:image-%=build-%)
all-images: $(IMAGES) ;

image-pgo-base: build/pgo-base/Dockerfile
	$(BUILDAH_CMD) build-using-dockerfile \
		--tag $(BUILDAH_TRANSPORT)$(PGO_IMAGE_PREFIX)/pgo-base:$(PGO_IMAGE_TAG) \
		--build-arg 'BASEOS=$(PGO_BASEOS)' \
		--build-arg 'DOCKERBASEREGISTRY=$(DOCKERBASEREGISTRY)' \
		--build-arg 'PACKAGER=$(PACKAGER)' \
		--build-arg 'PG_FULL=$(PGO_PG_FULLVERSION)' \
		--build-arg 'PGVERSION=$(PGO_PG_VERSION)' \
		--build-arg 'RELVER=$(PGO_VERSION)' \
		--file $< --format docker --layers $(BUILDAH_BUILD_ARGS) .

$(OTHER_IMAGES): image-%: build/%/Dockerfile build-% image-pgo-base
	$(BUILDAH_CMD) build-using-dockerfile \
		--tag $(BUILDAH_TRANSPORT)$(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG) \
		--build-arg 'ANSIBLE_VERSION=$(ANSIBLE_VERSION)' \
		--build-arg 'BACKREST_VERSION=$(PGO_BACKREST_VERSION)' \
		--build-arg 'BASEOS=$(PGO_BASEOS)' \
		--build-arg 'BASEVER=$(PGO_VERSION)' \
		--build-arg 'DFSET=$(DFSET)' \
		--build-arg 'PACKAGER=$(PACKAGER)' \
		--build-arg 'PGVERSION=$(PGO_PG_VERSION)' \
		--build-arg 'PREFIX=$(BUILDAH_TRANSPORT)$(PGO_IMAGE_PREFIX)' \
		--file $< --format docker --layers $(BUILDAH_BUILD_ARGS) .


#======== Utility =======
cli-docs:
	rm docs/content/pgo-client/reference/*.md
	cd docs/content/pgo-client/reference && go run ../../../../cmd/pgo/generatedocs.go
	sed -e '1,5 s|^title:.*|title: "pgo Client Reference"|' \
		docs/content/pgo-client/reference/pgo.md > \
		docs/content/pgo-client/reference/_index.md
	rm docs/content/pgo-client/reference/pgo.md

clean: clean-deprecated
	rm -f bin/apiserver
	rm -f bin/postgres-operator
	rm -f bin/pgo bin/pgo-mac bin/pgo.exe
	rm -f bin/pgo-backrest/pgo-backrest
	rm -f bin/pgo-rmdata/pgo-rmdata
	rm -f bin/pgo-scheduler/pgo-scheduler
	[ -z "$$(ls hack/tools)" ] || rm hack/tools/*

clean-deprecated:
	@# packages used to be downloaded into the vendor directory
	[ ! -d vendor ] || rm -r vendor
	@# executables used to be compiled into the $GOBIN directory
	[ ! -n '$(GOBIN)' ] || rm -f $(GOBIN)/postgres-operator $(GOBIN)/apiserver $(GOBIN)/*pgo
	[ ! -d bin/postgres-operator ] || rm -r bin/postgres-operator

push: $(IMAGES:image-%=push-%) ;
push-%:
	$(IMG_PUSHER_PULLER) push $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)

pull: $(IMAGES:image-%=pull-%) ;
pull-%:
	$(IMG_PUSHER_PULLER) pull $(PGO_IMAGE_PREFIX)/$*:$(PGO_IMAGE_TAG)

release:  linuxpgo macpgo winpgo
	rm -rf $(RELTMPDIR) $(RELFILE)
	mkdir $(RELTMPDIR)
	cp -r $(PGOROOT)/examples $(RELTMPDIR)
	cp -r $(PGOROOT)/deploy $(RELTMPDIR)
	cp -r $(PGOROOT)/conf $(RELTMPDIR)
	cp bin/pgo $(RELTMPDIR)
	cp bin/pgo-mac $(RELTMPDIR)
	cp bin/pgo.exe $(RELTMPDIR)
	cp $(PGOROOT)/examples/pgo-bash-completion $(RELTMPDIR)
	tar czvf $(RELFILE) -C $(RELTMPDIR) .

generate:
	GOBIN='$(CURDIR)/hack/tools' ./hack/update-codegen.sh
