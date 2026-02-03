# Erstellen von Middleware mit benutzerdefinierten Plugins

Traefik-Middlewares können benutzerdefinierte Plugins laden.

## So funktioniert es

Traefik-Plugins sind in Go geschrieben. Die Plugin-Dateien müssen aus einer ConfigMap in den Traefik-Container gemountet werden. Zusätzlich zum
Quellcode benötigt jedes Plugin eine Datei namens ``.traefik.yml``. Diese Datei enthält die Konfiguration und Informationen zum Plugin.
Ein Beispiel befindet sich unter [rewritebody](k8s/helm/files/rewritebody/.traefik.yml).
Auf diese Weise erstellte Plugins müssen nicht extern heruntergeladen werden.

## Erstellen eines Plugins

1. Den Quellcode des Plugins und die Datei ``.traefik.yml`` in ``k8s/helm/files/<plugin-name>`` speichern.
2. Den Quellcode des Plugins zu einer ConfigMap hinzufügen.

    ```yaml
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: <plugin-name>
        data:
        {{- $files := .Files. Glob „files/<plugin-name>/**“ -}}
        {{- range $path, $file := $files }}
          {{ $path | base | quote }}: |-
        {{ $file | toString | indent 4 }}
        {{- end }}
       
    ```
   
3. Die ConfigMap in das Traefik Deployment mounten

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
   