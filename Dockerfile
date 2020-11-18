FROM centos:7

ENV GIN_MODE=release

COPY config-example.yaml  /config.yaml
COPY bin/helm-wrapper /helm-wrapper

CMD [ "/helm-wrapper" ]
