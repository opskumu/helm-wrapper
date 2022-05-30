FROM centos:7

ENV GIN_MODE=release

COPY config-docker.yaml  /config.yaml
COPY bin/helm-wrapper /helm-wrapper

CMD [ "/helm-wrapper" ]
