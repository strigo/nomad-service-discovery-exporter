FROM scratch

LABEL org.opencontainers.image.source=https://github.com/strigo/nomad-service-discovery-exporter
LABEL org.opencontainers.image.description="A Prometheus exporter that reports the health status of services in Nomad's native service discovery"
LABEL org.opencontainers.image.licenses=MIT

COPY nomad-service-discovery-exporter /nomad-service-discovery-exporter

ENTRYPOINT ["/nomad-service-discovery-exporter"]