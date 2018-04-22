# monitor-power

A golang daemon to graph the computer power usage into prometheus or expvar.

# Demo

```sh
# prometheus
DOCKERIP=`ip addr | grep docker -A 1 | grep inet | cut -f 1 -d / | cut -b 10- | tr -d "\n"`
cat ./prometheus.tpl.yml | sed -e "s/localhost:9096/$DOCKERIP:9096/" > prometheus.yml
docker run -d -p 9090:9090 -v `pwd`/prometheus.yml:/etc/prometheus/prometheus.yml -name "monitor-power/prometheus" prom/prometheus
xdg-open http://localhost:9090/targets
xdg-open http://localhost:9090/graph?g0.range_input=5m&g0.expr=watts_now&g0.tab=0

# expvar
expvarmon -ports="9096" -vars="current_now,voltage_now,watts_now,\
mem:memstats.Alloc,mem:memstats.Sys\
,mem:memstats.HeapAlloc,mem:memstats.HeapInuse,\
duration:memstats.PauseNs,duration:memstats.PauseTotalNs" -i 250ms

# monitor
go run *.go
```

# Usage

```sh
$ go run *.go -h
Usage
  monitor-power -collect 2s
  sudo ./monitor-power install -collect 2s -os whatever -port :6666
  sudo ./monitor-power (start|stop|satus|remove)
  sudo journalctl -f -u monitor-power.service

Options of monitor-power:
  -collect duration
    	Time interval of metrics collect. (default 1s)
  -http string
    	http listen address interface of the stats. (default ":9096")
  -os string
    	the underlying os if not detected automatically, it is not. (default "fedora")
```
