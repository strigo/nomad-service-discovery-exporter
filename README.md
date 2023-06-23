Nomad Service Discovery Exporter
================================

A Prometheus exporter that reports the health status of services in [Nomad's
native service discovery][1].

This is a temporary solution until [nomad/16602][2] is fixed/implemented.


Compatibility
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


[1]: https://developer.hashicorp.com/nomad/docs/networking/service-discovery
[2]: https://github.com/hashicorp/nomad/issues/16602
