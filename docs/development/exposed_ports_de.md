# Exposed Ports

Traefik erlaubt keine dynamische Freigabe von Ports. Aus diesem Grund wird die Konfiguration über ConfigMaps vorgenommen. 
Dafür existiert zum einen die ConfigMap [ces-gateway-config](../../k8s/helm/templates/ces-gateway-config.yaml).
Diese ConfigMap dient zur initialen Konfiguration. Sie wird mit `k8s-ces-gateway` über Helm deployt. 
Helm überschreibt jede Änderungen an dieser ConfiMap bei einem Upgrade. Aus diesem Grund kann diese ConfigMap nicht 
zur Verwaltung der Exposed Ports verwendet werden. 
Dafür wird durch den `k8s-dogu-operator` eine ConfigMap mit dem Label `k8s.cloudogu.com/component.config: k8s-ces-gateway` erstellt.
Diese ConfigMap wird durch den `k9s-component-operator` gewatched und `k8s-ces-gateway` bei Änderungen gereconciled.
Der `k8s-dogu-operator` schreibt die Exposed Ports der Dogus in die ConfigMap, so wie sie vorher in der `values.yaml`aufgeführt waren.
Bei einer Änderung an der ConfigMap wird `k8s-ces-gateway` neu gestartet und die Ports sind freigegeben.