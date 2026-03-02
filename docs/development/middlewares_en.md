# Middlewares

The k8s-ces-gateway uses multiple Traefik middlewares to add functionality. The middlewares are configured in 
``k8s/helm/templates`` and added through the ``values.yaml`` file or the ``middleware-chain.yaml`` file.

## Middleware List
* Accept-Enconding
  * sets the Accept-Encoding header to ``identity`` to avoid gzip compression and add custom content to the response
* RewriteBody
  * Writes custom scripts and css (warp menu and whitelabeling) to the response body
  * custom middleware
* Compress
  * compresses the response body using gzip after custom content was added
* ErrorPages
  * use the error pages from k8s-ces-assets