package main

import (
	"fmt"
	"github.com/dlopes7/aix-prometheus-exporter/collector"
	"github.com/dlopes7/aix-prometheus-exporter/https"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"sort"
)

type handler struct {
	unfilteredHandler       http.Handler
	exporterMetricsRegistry *prometheus.Registry
	maxRequests             int
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["collect[]"]
	log.Debugln("collect query:", filters)

	if len(filters) == 0 {
		h.unfilteredHandler.ServeHTTP(w, r)
		return
	}
	filteredHandler, err := h.innerHandler(filters...)
	if err != nil {
		log.Warnln("Couldn't create filtered metrics handler:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create filtered metrics handler: %s", err)))
		return
	}
	filteredHandler.ServeHTTP(w, r)
}

func (h *handler) innerHandler(filters ...string) (http.Handler, error) {
	ac, err := collector.NewAIXCollector(filters...)
	if err != nil {
		return nil, fmt.Errorf("couldn't create collector: %s", err)
	}

	if len(filters) == 0 {
		log.Infof("Enabled collectors:")
		collectors := []string{}
		for n := range ac.Collectors {
			collectors = append(collectors, n)
		}
		sort.Strings(collectors)
		for _, n := range collectors {
			log.Infof(" - %s", n)
		}
	}

	r := prometheus.NewRegistry()
	r.MustRegister(version.NewCollector("aix_exporter"))
	if err := r.Register(ac); err != nil {
		return nil, fmt.Errorf("couldn't register node collector: %s", err)
	}
	handler := promhttp.HandlerFor(
		prometheus.Gatherers{h.exporterMetricsRegistry, r},
		promhttp.HandlerOpts{
			ErrorLog:            log.NewErrorLogger(),
			ErrorHandling:       promhttp.ContinueOnError,
			MaxRequestsInFlight: h.maxRequests,
			Registry:            h.exporterMetricsRegistry,
		},
	)

	handler = promhttp.InstrumentMetricHandler(
		h.exporterMetricsRegistry, handler,
	)

	return handler, nil
}

func newHandler(maxRequests int) *handler {
	h := &handler{
		exporterMetricsRegistry: prometheus.NewRegistry(),
		maxRequests:             maxRequests,
	}

	h.exporterMetricsRegistry.MustRegister(
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		prometheus.NewGoCollector(),
	)

	if innerHandler, err := h.innerHandler(); err != nil {
		log.Fatalf("Couldn't create metrics handler: %s", err)
	} else {
		h.unfilteredHandler = innerHandler
	}
	return h
}

func main() {

	var (
		listenAddress = kingpin.Flag(
			"web.listen-address",
			"Address on which to expose metrics and web interface.",
		).Default(":9100").String()

		metricsPath = kingpin.Flag(
			"web.telemetry-path",
			"Path under which to expose metrics.",
		).Default("/metrics").String()

		maxRequests = kingpin.Flag(
			"web.max-requests",
			"Maximum number of parallel scrape requests. Use 0 to disable.",
		).Default("40").Int()

		configFile = kingpin.Flag(
			"web.config",
			"Path to config yaml file that can enable TLS or authentication.",
		).Default("").String()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("aix_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting aix_exporter")
	http.Handle(*metricsPath, newHandler(*maxRequests))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Node Exporter</title></head>
			<body>
			<h1>Node Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	server := &http.Server{Addr: *listenAddress}
	if err := https.Listen(server, *configFile); err != nil {
		log.Fatal(err)
	}
}
