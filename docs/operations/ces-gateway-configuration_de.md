# Konfigurationsoptionen

## Read-Timeout

Standardmäßig verfügt das dem ces-gateway-Komponente zugrundeliegende Traefik über ein Lese-Timeout (Read Timeout). 
Das bedeutet, dass eine aktive Verbindung, die Daten an den Server überträgt, nach Ablauf dieser Zeit beendet wird. 
Im Gateway ist dieser Wert standardmäßig auf 5 Minuten eingestellt. Wenn eine Installation ein anderes Lese-Timeout 
erfordert, kann dies über das Feld `spec.valuesYamlOverwrite` konfiguriert werden. Dabei ist das Ändern von 2 Timeouts 
erforderlich, eine für die unverschlüsselte und eine für die unverschlüsselte Übertragung. 

Bei der Formatierung der Zeitdauer muss auf den Wert die Zeiteinheit folgen, wobei diese auch kombiniert werden können. 
Beispiele:
- 300s
- 10m
- 1h30m

Um das Timeout für Lesevorgänge vollständig zu deaktivieren und unbegrenzte Übertragungszeiten zu ermöglichen, kann der
Wert auf 0 gesetzt werden.

Hier ein Beispiel, bei dem die beiden Timeouts hochgesetzt werden:

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