#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

echo "                                     ./////,                    "
echo "                                 ./////==//////*                "
echo "                                ////.  ___   ////.              "
echo "                         ,**,. ////  ,////A,  */// ,**,.        "
echo "                    ,/////////////*  */////*  *////////////A    "
echo "                   ////'        \VA.   '|'   .///'       '///*  "
echo "                  *///  .*///*,         |         .*//*,   ///* "
echo "                  (///  (//////)**--_./////_----*//////)   ///) "
echo "                   V///   '°°°°      (/////)      °°°°'   ////  "
echo "                    V/////(////////\. '°°°' ./////////(///(/'   "
echo "                       'V/(/////////////////////////////V'      "

export POD_NAMESPACE=ecosystem
# Start nginx
echo "[nginx] starting nginx service..."
/nginx-ingress-controller \
  --publish-service="${POD_NAMESPACE}"/nginx-ingress \
  --election-id=ingress-controller-leader \
  --controller-class=k8s.io/nginx-ingress \
  --ingress-class=k8s-ecosystem-ces-service \
  --configmap="${POD_NAMESPACE}"/k8s-ces-gateway-nginx-ingress \
  --validating-webhook=:8443 \
  --validating-webhook-certificate=/usr/local/certificates/cert \
  --validating-webhook-key=/usr/local/certificates/key \
  --default-ssl-certificate="${POD_NAMESPACE}"/ecosystem-certificate \
  --watch-namespace="${POD_NAMESPACE}" \
  --tcp-services-configmap="${POD_NAMESPACE}"/k8s-ces-gateway-tcp-services \
  --udp-services-configmap="${POD_NAMESPACE}"/k8s-ces-gateway-udp-services