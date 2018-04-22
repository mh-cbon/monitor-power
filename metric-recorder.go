package main

import (
	"expvar"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

//MetricsRecorder is a provider of observer and gauges.
type MetricsRecorder interface {
	Counter(name string, help ...string) StatObserver
	Duration(name string, help ...string) StatObserver
	Gauge(name string, help ...string) StatGauge
}

//StatGauge is something that records values of int, durations, floats.
type StatGauge interface {
	Set(float64)
	Add(float64)
}

//StatObserver is something that observe values of int, durations, floats.
type StatObserver interface {
	Observe(float64)
}

//durationObserver is the ExpVar implementaton of the statObserver when talking about duration.
type durationObserver struct {
	*expvar.Int
	*sync.Mutex
	tick           time.Time
	values         []float64
	events         float64
	reduceInterval time.Duration
}

func newDurationObserver(name string, reduceInterval time.Duration) *durationObserver {
	return &durationObserver{
		expvar.NewInt(name),
		&sync.Mutex{},
		time.Now(),
		[]float64{},
		0,
		reduceInterval,
	}
}

func (e *durationObserver) Observe(value float64) {
	e.Lock()
	defer e.Unlock()
	if time.Since(e.tick) > e.reduceInterval {
		e.tick = time.Now()
		if len(e.values) > 10 {
			e.values = append(e.values[:0], e.values[len(e.values)-1:]...)
		}
	}
	e.values = append(e.values, value)
	t := len(e.values)
	if t > 2 {
		total := time.Duration(0)
		for _, v := range e.values {
			total += time.Duration(v)
		}
		a := total / time.Duration(t)
		e.Set(int64(a))
	}
}

//counterObserver is the ExpVar implementaton of the statObserver when talking about counter values.
type counterObserver struct {
	*expvar.Float
	*sync.Mutex
	tick   time.Time
	values []float64
	events float64
}

func newCounterObserver(name string) *counterObserver {
	return &counterObserver{
		expvar.NewFloat(name),
		&sync.Mutex{},
		time.Now(),
		[]float64{},
		0,
	}
}

func (e *counterObserver) Observe(value float64) {
	e.Lock()
	defer e.Unlock()
	j := len(e.values)
	if time.Since(e.tick) > time.Second {
		e.values = append(e.values, e.events)
		e.tick = time.Now()
		e.events = 0
		if j > 20 {
			e.values = append(e.values[:0], e.values[len(e.values)-2:]...)
		}
	}
	e.events += value
	if j > 2 {
		total := 0.0
		for _, v := range e.values {
			total += float64(v)
		}
		x := total / float64(j)
		e.Set(x)
	} else {
		e.Add(1)
	}
}

//Expvar implementaton of the MetricsRecorder.
type Expvar struct {
	ReduceInterval time.Duration
}

//Counter returns an observer of values.
func (e Expvar) Counter(name string, help ...string) StatObserver { return newCounterObserver(name) }

//Duration returns an observer of durations.
func (e Expvar) Duration(name string, help ...string) StatObserver {
	return newDurationObserver(name, e.ReduceInterval)
}

//Gauge returns an observer of floats.
func (e Expvar) Gauge(name string, help ...string) StatGauge { return expvar.NewFloat(name) }

var defaultHelp = "no description provided"

//Prometheus implementaton of the MetricsRecorder.
type Prometheus struct{}

//Counter returns an observer of values.
func (p Prometheus) Counter(name string, help ...string) StatObserver {
	h := defaultHelp
	if len(help) > 0 {
		h = help[0]
	}
	ret := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    name,
		Help:    h,
		Buckets: prometheus.LinearBuckets(20, 5, 5),
	})
	prometheus.MustRegister(ret)
	return ret
}

//Duration returns an observer of durations.
func (p Prometheus) Duration(name string, help ...string) StatObserver {
	h := defaultHelp
	if len(help) > 0 {
		h = help[0]
	}
	ret := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    name,
		Help:    h,
		Buckets: prometheus.LinearBuckets(20, 5, 5),
	})
	prometheus.MustRegister(ret)
	return ret
}

//Gauge returns an observer of floats.
func (p Prometheus) Gauge(name string, help ...string) StatGauge {
	h := defaultHelp
	if len(help) > 0 {
		h = help[0]
	}
	ret := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: h,
	})
	prometheus.MustRegister(ret)
	return ret
}

type durationMulti []StatObserver

func (d durationMulti) Observe(some float64) {
	for _, observer := range d {
		observer.Observe(some)
	}
}

type counterMulti []StatObserver

func (c counterMulti) Observe(some float64) {
	for _, observer := range c {
		observer.Observe(some)
	}
}

type gaugeMulti []StatGauge

func (g gaugeMulti) Set(some float64) {
	for _, observer := range g {
		observer.Set(some)
	}
}
func (g gaugeMulti) Add(some float64) {
	for _, observer := range g {
		observer.Add(some)
	}
}

//MultiRecorder of MetricsRecorder
type MultiRecorder map[string]MetricsRecorder

//Counter to multiple backends.
func (m MultiRecorder) Counter(name string, help ...string) StatObserver {
	ret := counterMulti{}
	for _, v := range m {
		ret = append(ret, v.Counter(name, help...))
	}
	return ret
}

//Duration to multiple backends.
func (m MultiRecorder) Duration(name string, help ...string) StatObserver {
	ret := durationMulti{}
	for _, v := range m {
		ret = append(ret, v.Duration(name, help...))
	}
	return ret
}

//Gauge to multiple backends.
func (m MultiRecorder) Gauge(name string, help ...string) StatGauge {
	ret := gaugeMulti{}
	for _, v := range m {
		ret = append(ret, v.Gauge(name, help...))
	}
	return ret
}
