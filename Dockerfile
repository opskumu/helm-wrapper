FROM centos:7

COPY config-example.yaml  /config.yaml
COPY helm-wrapper /helm-wrapper

CMD [ "/helm-wrapper" ]
