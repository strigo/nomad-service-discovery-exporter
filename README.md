Nomad Service Health Exporter
=============================
Exports the health status of services registered to the native Nomad [service
discovery](https://developer.hashicorp.com/nomad/docs/networking/service-discovery).

This is a temporary solution until [nomad/issues/16602](https://github.com/hashicorp/nomad/issues/16602) is implemented.


Compatability
-------------
The exporter uses an undocumented API endpoint which might change/break between
new releases. This exporter was tested against Nomad v1.5.x.


Command line flags
------------------
See `-help` for details.


Available Metrics
-----------------
* `nomad_services`: The total number of services registered.
* `nomad_services_health`: Service health status.
