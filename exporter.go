package main

import (
	"context"
	"log"
	"sync"
	"time"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	CheckSuccess = "success"
	CheckFailure = "failure"
	CheckPending = "pending"
)

var (
	metricServicesTotal = metricInfo{
		prometheus.NewDesc(
			"nomad_services",
			"Number of services registers to Nomad native service discovery",
			[]string{"namespace"},
			nil),
		prometheus.GaugeValue,
	}

	metricServicesHealth = metricInfo{
		prometheus.NewDesc(
			"nomad_services_health",
			"Health status of a service registered to Nomad Service Discovery",
			[]string{"namespace", "job_id", "task_name", "service_name", "check_name", "check_id", "status"},
			nil),
		prometheus.GaugeValue,
	}
)

type metricInfo struct {
	Desc *prometheus.Desc
	Type prometheus.ValueType
}

type Exporter struct {
	Config       *ExporterConfig
	client       *nomad.Client
	queryOptions *nomad.QueryOptions

	mutex        sync.RWMutex
	cache        sync.Map
	limit        chan struct{}

	totalErrs prometheus.Counter
}

type ExporterConfig struct {
	Address   string
	Region    string
	Namespace string
	SecretID  string

	Duration    time.Duration
	Parallelism int
	AllowStale  bool
}


func New(config *ExporterConfig) (*Exporter, error) {
	client, err := nomad.NewClient(&nomad.Config{
		Address:  config.Address,
		Region:   config.Region,
		SecretID: config.SecretID,
		Namespace: config.Namespace,
	})
	if err != nil {
		return nil, err
	}

	exporter := Exporter{
		client:       client,
		queryOptions: &nomad.QueryOptions{AllowStale: config.AllowStale},
		limit:        make(chan struct{}, config.Parallelism),
		Config:       config,

		totalErrs: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nomad_services_api_errors_total",
			Help: "Number of scrapes that resulted with one or more API errors from Nomad",
		}),
	}

	return &exporter, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricServicesTotal.Desc
	ch <- metricServicesHealth.Desc
	ch <- e.totalErrs.Desc()
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.cache = sync.Map{}

	// common context for all the requests - cancel everything when deadline reached
	ctx, cancel := context.WithTimeout(context.Background(), e.Config.Duration)
	defer cancel()
	e.queryOptions = e.queryOptions.WithContext(ctx)

	err := e.collectServices(ch)
	if err != nil {
		log.Println("scrape error ", err)
		e.totalErrs.Inc()
	}

	ch <- e.totalErrs
}

func (e *Exporter) collectServices(ch chan<- prometheus.Metric) error {
	e.limit <- struct{}{}
	listStub, _, err := e.client.Services().List(e.queryOptions)
	<-e.limit
	if err != nil {
		return err
	}

	errg := new(errgroup.Group)

	for _, stub := range listStub {
		stub := stub
		ch <- prometheus.MustNewConstMetric(
			metricServicesTotal.Desc, metricServicesTotal.Type, float64(len(stub.Services)), stub.Namespace,
		)

		errg.Go(func() error {
			return e.collectNamespace(stub.Services, ch)
		})
	}
	return errg.Wait()
}

func (e *Exporter) collectNamespace(services []*nomad.ServiceRegistrationStub, ch chan<- prometheus.Metric) error {
	errg := new(errgroup.Group)

	for _, svc := range services {
		svc := svc
		errg.Go(func() error {
			return e.collectService(svc.ServiceName, ch)
		})
	}

	return errg.Wait()
}

func (e *Exporter) collectService(svc string, ch chan<- prometheus.Metric) error {
	errg := new(errgroup.Group)

	e.limit <- struct{}{}
	registrations, _, err := e.client.Services().Get(svc, e.queryOptions)
	<-e.limit
	if err != nil {
		return err
	}

	for _, r := range registrations {
		// same allocID can exists for multiple registrations
		// avoid scanning and reporting metrics for same allocs (prometheus will complain)
		r := r
		if _, exists := e.cache.LoadOrStore(r.AllocID, struct{}{}); exists {
			continue
		}

		errg.Go(func() error {
			return e.collectAllocation(r, ch)
		})
	}

	return errg.Wait()
}

func (e *Exporter) collectAllocation(reg *nomad.ServiceRegistration, ch chan<- prometheus.Metric) error {
	e.limit <- struct{}{}
	statuses, err := e.client.Allocations().Checks(reg.AllocID, e.queryOptions)
	<-e.limit
	if err != nil {
		log.Printf("unable to scrape allocation %v: %v\n", reg.AllocID, err)
		return err
	}

	for _, status := range statuses {
		// two types of checks exists: readiness and healthiness, we only care about the later
		if status.Mode != "healthiness" {
			continue
		}

		var healthy, failure, pending float64
		switch status.Status {
		case CheckSuccess:
			healthy = 1
		case CheckFailure:
			failure = 1
		case CheckPending:
			pending = 1
		}

		ch <- prometheus.MustNewConstMetric(
			metricServicesHealth.Desc, metricServicesHealth.Type, healthy, reg.Namespace, reg.JobID, status.Task, status.Service, status.Check, status.ID, CheckSuccess,
		)

		ch <- prometheus.MustNewConstMetric(
			metricServicesHealth.Desc, metricServicesHealth.Type, failure, reg.Namespace, reg.JobID, status.Task, status.Service, status.Check, status.ID, CheckFailure,
		)

		ch <- prometheus.MustNewConstMetric(
			metricServicesHealth.Desc, metricServicesHealth.Type, pending, reg.Namespace, reg.JobID, status.Task, status.Service, status.Check, status.ID, CheckPending,
		)
	}

	return nil
}
