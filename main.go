package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)


var (
	BindAddr       = flag.String("bind", ":9100", "Bind listen socket to this address")
	MetricsPath    = flag.String("path", "/metrics", "Path to export metrics at")
	NomadAddr      = flag.String("addr", "http://127.0.0.1:4646", "Nomad address")
	NomadNamespace = flag.String("namespace", "", "Nomad namespace")
	NomadRegion    = flag.String("region", "", "Nomad region (defaults to the agent's region)")
	NomadStale     = flag.Bool("stale", true, "Allow stale read results (disable with -no-stale)")

	MaxDuration = flag.Duration("duration", time.Second*11, "Max scrape time limit (seconds)")
	Parallelism = flag.Int("parallelism", 32, "Max amount of parallel outgoing requests to Nomad HTTP API")

	version = "0.0.0-devel"
)

func main() {
	log.Printf("Starting nomad-service-discovery-exporter v%v", version)

	flag.Parse()

	exporter, err := New(
		&ExporterConfig{
			Address:     *NomadAddr,
			Region:      *NomadRegion,
			Namespace:   *NomadNamespace,
			Duration:    *MaxDuration,
			AllowStale:  *NomadStale,
			Parallelism: *Parallelism,
		})
	if err != nil {
		log.Fatal("Unable to create client: ", err)
	}

	prometheus.MustRegister(exporter)

	http.Handle(*MetricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
		<head><title>Nomad Service Discovery Exporter</title></head>
		<body>
		<h1>Nomad Service Discovery Exporter</h1>
		<p><a href='` + *MetricsPath + `'>Metrics</a></p>
		</body>
		</html>`))
	})
	log.Printf("Metrics available on: %v%v", *BindAddr, *MetricsPath)

	log.Fatal(http.ListenAndServe(*BindAddr, nil))
}
