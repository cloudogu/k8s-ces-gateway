FROM registry.k8s.io/ingress-nginx/controller:v1.12.1

LABEL maintainer="hello@cloudogu.com" \
      NAME="k8s-ces-gateway" \
      VERSION="0.0.1"

ENV INGRESS_USER=www-data

USER root

# copy files
COPY resources /
COPY k8s /k8s

# inject custom config into template
RUN /injectNginxConfig.sh

RUN apk update && apk upgrade && apk del curl

USER www-data

# Volumes are used to avoid writing to containers writable layer https://docs.docker.com/storage/
# Compared to the bind mounted volumes we declare in the dogu.json,
# the volumes declared here are not mounted to the dogu if the container is destroyed/recreated,
# e.g. after a dogu upgrade
VOLUME ["/etc/nginx/conf.d", "/var/log/nginx"]

# Define working directory.
WORKDIR /etc/nginx

# HEALTHCHECK CMD doguctl healthy nginx || exit 1

# Expose ports.
EXPOSE 80
EXPOSE 443

# Define default command.
ENTRYPOINT ["/startup.sh"]
