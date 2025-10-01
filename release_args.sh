#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

componentTemplateFile=k8s/helm/component-patch-tpl.yaml
ingressNginxTempChart="/tmp/ingress-nginx"
ingressNginxTempValues="${ingressNginxTempChart}/values.yaml"
ingressNginxTempChartYaml="${ingressNginxTempChart}/Chart.yaml"

cesGatewayValues="k8s/helm/values.yaml"

# this function will be sourced from release.sh and be called from release_functions.sh
update_versions_modify_files() {
  echo "Update helm dependencies"
  make helm-update-dependencies  > /dev/null

  # Extract ingress-nginx chart
  local ingressNginxVersion
  ingressNginxVersion=$(yq '.dependencies[] | select(.name=="ingress-nginx").version' < "k8s/helm/Chart.yaml")
  local ingressNginxPackage
  ingressNginxPackage="k8s/helm/charts/ingress-nginx-${ingressNginxVersion}.tgz"

  echo "Extract ingress-nginx helm chart"
  tar -zxvf "${ingressNginxPackage}" -C "/tmp" > /dev/null

  echo "Set images in component patch template"

  local controllerRegistry
  local controllerRepo
  local controllerTag
  controllerRegistry=$(yq '.global.image.registry' < "${ingressNginxTempValues}")
  controllerRepo=$(yq '.controller.image.image' < "${ingressNginxTempValues}")
  controllerTag=$(yq '.controller.image.tag' < "${ingressNginxTempValues}")
  setAttributeInComponentPatchTemplate ".values.images.controller" "${controllerRegistry}/${controllerRepo}:${controllerTag}"

  local webhookRepo
  webhookRepo=$(yq '.controller.admissionWebhooks.patch.image.image' < "${ingressNginxTempValues}")
  webhookTag=$(yq '.controller.admissionWebhooks.patch.image.tag' < "${ingressNginxTempValues}")
  setAttributeInComponentPatchTemplate ".values.images.webhook" "${controllerRegistry}/${webhookRepo}:${webhookTag}"

  rm -rf ${ingressNginxTempChart}
}

setAttributeInComponentPatchTemplate() {
  local key="${1}"
  local value="${2}"

  yq -i "${key} = \"${value}\"" "${componentTemplateFile}"
}

update_versions_stage_modified_files() {
  git add "${componentTemplateFile}"
}
