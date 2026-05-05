# Exposed Ports

Traefik does not support dynamic port exposure. For this reason, configuration is handled via ConfigMaps.
The ConfigMap [ces-gateway-config](../../k8s/helm/templates/ces-gateway-config.yaml) is provided for this purpose.
This ConfigMap is used for the initial configuration. It is deployed via Helm using `k8s-ces-gateway`.
Helm overwrites any changes to this ConfigMap during an upgrade. For this reason, this ConfigMap cannot
be used to manage the exposed ports.
Instead, the `k8s-dogu-operator` creates a ConfigMap with the label `k8s.cloudogu.com/component.config: k8s-ces-gateway`.
This ConfigMap is watched by the `k9s-component-operator` and `k8s-ces-gateway` is reconciled when changes occur.
The `k8s-dogu-operator` writes the Dogus’ exposed ports to the ConfigMap, exactly as they were previously listed in `values.yaml`.
When a change is made to the ConfigMap, `k8s-ces-gateway` is restarted and the ports are exposed.