#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

componentTemplateFile=k8s/helm/component-patch-tpl.yaml
cesGatewayValues="k8s/helm/values.yaml"

# this function will be sourced from release.sh and be called from release_functions.sh
update_versions_modify_files() {
  echo "Update helm dependencies"
  make helm-update-dependencies  > /dev/null

  local traefikRegistry
  local traefikRepo
  local traefikTag
  traefikRegistry=$(.bin/yq '.traefik.image.registry' < "${cesGatewayValues}")
  traefikRepo=$(.bin/yq '.traefik.image.repository' < "${cesGatewayValues}")
  traefikTag=$(.bin/yq '.traefik.image.tag' < "${cesGatewayValues}")

  setAttributeInComponentPatchTemplate ".values.images.traefik" "${traefikRegistry}/${traefikRepo}:${traefikTag}"
}

setAttributeInComponentPatchTemplate() {
  local key="${1}"
  local value="${2}"

  .bin/yq -i "${key} = \"${value}\"" "${componentTemplateFile}"
}

update_versions_stage_modified_files() {
  git add "${componentTemplateFile}"
}
