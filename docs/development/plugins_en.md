# Creating middleware with custom plugins

Traefik middlewares can load custom plugins. 

## How it works

Traefik plugins are written in go. The plugin files have to be mounted in the Traefik container from a configmap. In addition to its 
source code, every plugin needs a ``.traefik.yml`` file. This file contains the plugin's configuration and information. 
See [rewritebody](k8s/helm/files/rewritebody/.traefik.yml) for an example.
Plugins created this way do not need to be downloaded externally.

## Creating a plugin

1. Save the plugins source code and ``.traefik.yml`` file in ``k8s/helm/files/<plugin-name>``.
2. Add the plugins source code to a configmap. 

    ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: <plugin-name>
    data:
    {{- $files := .Files.Glob "files/<plugin-name>/**" -}}
    {{- range $path, $file := $files }}
      {{ $path | base | quote }}: |-
    {{ $file | toString | indent 4 }}
    {{- end }}
    
    ```
   
3. Mount the configmap in the Traefik deployment

    ```yaml
    traefik:
      enabled: true
      installCRDs: true
      deployment:
        additionalVolumes:
          - name: <plugin-name>-src
            configMap:
              name: ces-gateway-traefik-plugin-<plugin-name>
    
        additionalVolumeMounts:
            - name: <plugin-name>-src
              mountPath: /plugins-local/src/github.com/traefik/plugin-<plugin-name>
              readOnly: true
    
      experimental:
        localPlugins:
          <plugin-name>:
            moduleName: github.com/traefik/plugin-<plugin-name>
            mountPath: /plugins-local/src/github.com/traefik/plugin-<plugin-name>
            type: localPath
            volumeName: <plugin-name>-src
    ```
   
