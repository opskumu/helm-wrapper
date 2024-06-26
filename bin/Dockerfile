FROM scratch

ENV GIN_MODE=release

COPY config.yaml  /config.yaml
COPY helm-wrapper /helm-wrapper

ENTRYPOINT [ "/helm-wrapper" ]