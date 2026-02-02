# CES-external Ingresses

Cluster administrators might want to use this gateway component not only for their CES but for other services as well.
This is possible by running these services in a separate namespace under a different domain than the EcoSystem.

## Prerequisites

In this tutorial, we assume that our ces-external service uses the domain name `foo.bar.invalid`.
If you run a local cluster, be sure to add it to your `/etc/hosts` file.

We also want our example application to run in the `ingress-test` namespace, which we have to create:
```shell
kubectl create ns ingress-test
```

## Set the scope of the gateway

By default, the scope of the gateway is enabled, and it's namespace is set to the namespace it is deployed in.
This means that if the gateway is deployed in the `ecosystem` namespace, only ingress resources in that namespace will
be used.

Here's an example on how to configure the gateway component to watch all namespaces:
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
    traefik:
      providers:
        kubernetesIngress:
          namespaces: []        
        kubernetesCRD:
          namespaces: []
  version: 3.0.0
```

`namespaceSelector` allows us to only watch namespaces with specific labels. The format is `foo=bar`.
An empty `namespaceSelector` means all namespaces will be watched.

## Create certificate

We need a certificate matching the domain of the CES-external services.
For testing in this example, you can generate a self-signed certificate with the following command:
```shell
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 3650  -nodes -subj "/C=XX/ST=StateName/L=CityName/O=CompanyName/OU=CompanySectionName/CN=foo.bar.invalid"
```

The certificate can then be applied to the cluster with the following command:
```shell
kubectl -n ingress-test create secret tls foo-tls --cert=cert.pem --key=key.pem
```

## Creating the example application

You can find the kubernetes resources for the example application in this directory.
Simply apply it like this:
```shell
kubectl -n ingress-test apply -f example-application.yaml
```

If you want to know more, take a look at the example application, all the things to look out for have been commented.