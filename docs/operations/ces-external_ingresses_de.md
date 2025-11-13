# CES-externe Ingresses

Cluster-Administratoren möchten diese Gateway-Komponente möglicherweise nicht nur für ihren CES, sondern auch für andere
Dienste verwenden. Dies ist möglich, indem diese Dienste in einem separaten Namespace unter einer anderen Domain als dem
EcoSystem betrieben werden.

## Voraussetzungen

In diesem Tutorial gehen wir davon aus, dass unser ces-external-Dienst den Domainnamen `foo.bar.invalid` verwendet.
Wenn Sie einen lokalen Cluster betreiben, fügen Sie diesen Namen in Ihre Datei `/etc/hosts` ein.

Außerdem soll unsere Beispielanwendung im Namespace `ingress-test` laufen, den wir zunächst anlegen müssen:
```shell
kubectl create ns ingress-test
```

## Geltungsbereich (Scope) des Gateways festlegen

Standardmäßig ist der Scope des Gateways aktiviert und sein Namespace ist auf den Namespace gesetzt, in dem es
bereitgestellt (deployed) ist. Das bedeutet: Wenn das Gateway im Namespace `ecosystem` bereitgestellt ist, werden nur
Ingress-Ressourcen in diesem Namespace verwendet.

Hier ist ein Beispiel, wie Sie die Gateway-Komponente so konfigurieren, dass alle Namespaces beobachtet werden:
```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: k8s-ces-gateway
  namespace: ecosystem
spec:
  name: k8s-ces-gateway
  namespace: k8s
  valuesYamlOverwrite: |
    ingress-nginx:
      controller:
        scope:
          enabled: false
          namespaceSelector: ""
  version: 1.0.3
```

`namespaceSelector` ermöglicht es, nur Namespaces mit bestimmten Labels zu beobachten. Das Format ist `foo=bar`.
Ein leerer `namespaceSelector` bedeutet, dass alle Namespaces beobachtet werden.

## Zertifikat erstellen

Wir benötigen ein Zertifikat, das zur Domain der CES-external-Dienste passt.
Zum Testen in diesem Beispiel können Sie mit folgendem Befehl ein selbstsigniertes Zertifikat erzeugen:
```shell
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 3650  -nodes -subj "/C=XX/ST=StateName/L=CityName/O=CompanyName/OU=CompanySectionName/CN=foo.bar.invalid"
```

Das Zertifikat kann anschließend mit dem folgenden Befehl im Cluster hinterlegt werden:
```shell
kubectl -n ingress-test create secret tls foo-tls --cert=cert.pem --key=key.pem
```

## Die Beispielanwendung erstellen

Die Kubernetes-Ressourcen für die Beispielanwendung finden Sie in diesem Verzeichnis.
Wenden Sie sie einfach so an:
```shell
kubectl -n ingress-test apply -f example-application.yaml
```

Wenn Sie mehr wissen möchten, schauen Sie sich die Beispielanwendung an.
Alle wichtigen Punkte, auf die zu achten ist, wurden kommentiert.
