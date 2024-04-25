FROM scratch

ENV GIN_MODE=release

#COPY config-example.yaml  /config.yaml
COPY bin/helm-wrapper /helm-wrapper

ENTRYPOINT [ "/helm-wrapper" ]