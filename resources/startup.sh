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

# Namespace auto-detekten, wenn nicht gesetzt
: "${POD_NAMESPACE:=$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)}"

# Standard-Ports (kannst du via Env überschreiben)
HTTP_PORT="${HTTP_PORT:-80}"
HTTPS_PORT="${HTTPS_PORT:-443}"
WEBHOOK_PORT="${WEBHOOK_PORT:-8443}"

# Optional: zusätzliche Flags über Env injizierbar (Helm-Values)
EXTRA_ARGS="${EXTRA_ARGS:-}"

# Start nginx
echo "[ingress-nginx] starting controller on :${HTTP_PORT}/:${HTTPS_PORT} …"

# echo '''
exec /nginx-ingress-controller \
  --controller-class=k8s.io/nginx-ingress \
  --ingress-class=k8s-ecosystem-ces-service \
  --publish-service="${POD_NAMESPACE}"/nginx-ingress \
  --configmap="${POD_NAMESPACE}"/k8s-ces-gateway-nginx-ingress \
  --election-id=ingress-controller-leader \
  --validating-webhook=":${WEBHOOK_PORT}" \
  --validating-webhook-certificate=/usr/local/certificates/cert \
  --validating-webhook-key=/usr/local/certificates/key \
  --default-ssl-certificate="${POD_NAMESPACE}"/ecosystem-certificate \
  --watch-namespace="${POD_NAMESPACE}" \
  --http-port="${HTTP_PORT}" \
  --https-port="${HTTPS_PORT}" \
  --tcp-services-configmap="${POD_NAMESPACE}"/tcp-services \
  --udp-services-configmap="${POD_NAMESPACE}"/udp-services \
${EXTRA_ARGS}
# '''
# sleep 60000

