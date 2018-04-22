// Package monitor-power simply reads data from
// linux proc files and store them under
// /metrics golang handler.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/takama/daemon"
)

type options struct {
	http            string
	os              string
	collectInterval time.Duration
}

func main() {

	var opts = options{}

	flag.StringVar(&opts.os, "os", "fedora", "the underlying os if not detected automatically, it is not.")
	flag.StringVar(&opts.http, "http", ":9096", "http listen address interface of the stats.")
	flag.DurationVar(&opts.collectInterval, "collect", time.Duration(time.Second), "Time interval of metrics collect.")

	flag.Parse()

	action := ""
	if flag.NArg() > 0 {
		action = flag.Arg(0)
	}

	context := context.Background()
	actions := map[string]func() error{
		"": func() error { return daemonized(context, opts) },
		"install": func() error {
			return queryService(func() (string, error) {
				return getService().Install(
					"-collect", fmt.Sprint(opts.collectInterval),
					"-http", opts.http,
					"-os", opts.os,
				)
			})
		},
		"start":  func() error { return queryService(getService().Start) },
		"stop":   func() error { return queryService(getService().Stop) },
		"status": func() error { return queryService(getService().Manage) },
		"remove": func() error { return queryService(getService().Remove) },
	}

	if run, ok := actions[action]; ok {
		run()
	} else {
		panic("action is unknown " + action)
	}

}

func daemonized(context context.Context, opts options) error {

	// setup the promethus endpoint
	http.Handle("/metrics", promhttp.Handler())

	// start up metrics server. Notes that expvar adds the endpoint at init().
	go http.ListenAndServe(opts.http, nil)

	metricsRecorder := MultiRecorder{
		"expvar": Expvar{
			ReduceInterval: time.Second,
		},
		"prometheus": Prometheus{},
	}

	metrics := map[string]StatGauge{
		"current": metricsRecorder.Gauge("current_now"),
		"voltage": metricsRecorder.Gauge("voltage_now"),
		"watts":   metricsRecorder.Gauge("watts_now"),
	}

	osProviders := map[string]metricProvider{
		"fedora": fedoraProvider{},
	}

	var provider metricProvider
	if x, ok := osProviders[opts.os]; ok {
		provider = x
	} else {
		panic("os not found " + opts.os)
	}

	ticker := time.NewTicker(opts.collectInterval)

	go func() {
		collect := map[string]CollectedMetric{}
		for k := range metrics {
			collect[k] = CollectedMetric{"", nil}
		}
		for {
			<-ticker.C
			provider.Collect(collect)
			for k, v := range collect {
				if err := v.Err; err != nil {
					log.Printf("failed to parse metric %q, err=%v\n", k, err)
					continue
				}
				x, err := strconv.ParseFloat(v.Value, 64)
				if err != nil {
					log.Printf("failed to parse metric %q, err=%v\n", k, err)
					continue
				}
				metrics[k].Set(x)
				collect[k] = CollectedMetric{"", nil}
			}
			log.Printf("collected %v metrics\n", len(collect))
		}
	}()

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Kill)
	<-sig

	return nil
}

//    dependencies that are NOT required by the service, but might be used
var dependencies = []string{}

const (

	// name of the service
	name        = "monitor-power"
	description = "Monitor power usage"

	// service port
	port = ":9077"
)

func getService() *Service {
	srv, err := daemon.New(name, description, dependencies...)
	if err != nil {
		errlog.Println("Error: ", err)
		os.Exit(1)
	}
	return &Service{srv}
}

func queryService(handler func() (string, error)) error {
	status, err := handler()
	if err != nil {
		errlog.Println(status, "\nError: ", err)
	}
	fmt.Println(status)
	return err
}
