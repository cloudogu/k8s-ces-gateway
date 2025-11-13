# Set these to the desired values
ARTIFACT_ID=k8s-ces-gateway
VERSION=1.0.3

MAKEFILES_VERSION=10.2.0

include build/make/variables.mk
include build/make/self-update.mk
include build/make/clean.mk
include build/make/release.mk
include build/make/k8s-component.mk

ADDITIONAL_CLEAN=clean_charts

clean_charts:
	rm -rf ${HELM_SOURCE_DIR}/charts

.PHONY: k8s-ces-gateway-release
k8s-ces-gateway-release: ## Interactively starts the release workflow for k8s-ces-gateway
	@echo "Starting git flow release..."
	@build/make/release.sh k8s-ces-gateway