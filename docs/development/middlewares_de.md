# Middlewares

Das k8s-ces-Gateway verwendet mehrere Traefik-Middlewares, um Funktionen hinzuzufügen. Die Middlewares werden in
``k8s/helm/templates`` konfiguriert und über die Datei ``values.yaml`` oder die Datei ``middleware-chain.yaml`` hinzugefügt.

## Liste der Middlewares
* Accept-Enconding
    * Setzt den Accept-Encoding-Header auf „identity“, um eine gzip-Komprimierung zu vermeiden und benutzerdefinierte Inhalte zur Response hinzuzufügen.
* RewriteBody
    * Schreibt benutzerdefinierte Skripte und CSS (Warp-Menü und Whitelabeling) in die Response.
    * Benutzerdefinierte Middleware
* Compress
    * Komprimiert die Response mit gzip, nachdem benutzerdefinierte Inhalte hinzugefügt wurden.
* ErrorPages
    * Verwendet die Fehlerseiten aus k8s-ces-assets.
