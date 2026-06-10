# Configuration Options

## Read Timeout

By default, the Traefik underlying the ces-gateway component has a read timeout, meaning the time when an active 
connection that is transferring data to the server will be terminated. By default, this is set to 5 minutes in the 
gateway. If an installation requires a different read timeout, that can be accomplished by using the
`spec.valuesYamlOverwrite` field. Two values need to be set, one for the secure and one for the unsecure connection. 

The time duration format needs to be followed by the time unit, and these can be combined. Examples:
- 300s
- 10m
- 1h30m

To disable the timeout for reading operations and allow infinite transfer times, the value can be set to 0.  


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
      ports:
        web:
          transport:
            respondingTimeouts:
              readTimeout: 10m
        websecure:
          transport:
            respondingTimeouts:
              readTimeout: 10m
  version: 3.0.0
```