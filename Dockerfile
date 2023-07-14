FROM scratch

COPY nomad-service-health-exporter /nomad-service-health-exporter

ENTRYPOINT ["/nomad-service-health-exporter"]