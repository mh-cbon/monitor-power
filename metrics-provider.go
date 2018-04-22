package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

type metricProvider interface {
	Collect(dst map[string]CollectedMetric)
}

// CollectedMetric is the result of the collection of the metric on the underlying system.
type CollectedMetric struct {
	Value string
	Err   error
}

type fedoraProvider struct {
}

func (f fedoraProvider) Collect(dst map[string]CollectedMetric) {
	for k := range dst {
		if k == "voltage" {
			v, err := f.Voltage()
			dst[k] = CollectedMetric{v, err}
		} else if k == "current" {
			v, err := f.Current()
			dst[k] = CollectedMetric{v, err}
		} else if k == "watts" {
			v, err := f.Watts()
			dst[k] = CollectedMetric{v, err}
		}
	}
}
func (f fedoraProvider) Voltage() (string, error) {
	data, err := ioutil.ReadFile("/sys/class/power_supply/BAT0/voltage_now")
	ret := string(data)
	ret = strings.TrimSpace(ret)
	return ret, err
}
func (f fedoraProvider) Current() (string, error) {
	data, err := ioutil.ReadFile("/sys/class/power_supply/BAT0/current_now")
	ret := string(data)
	ret = strings.TrimSpace(ret)
	return ret, err
}
func (f fedoraProvider) UEvents() (map[string]string, error) {
	ret := map[string]string{}
	data, err := ioutil.ReadFile("/sys/class/power_supply/BAT0/uevent")
	if err != nil {
		return ret, err
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(data))
	for scanner.Scan() {
		b := scanner.Bytes()
		s := strings.Split(strings.TrimSpace(string(b)), "=")
		ret[s[0]] = s[1]
	}
	if err = scanner.Err(); err != nil {
		return ret, err
	}
	return ret, err
}
func (f fedoraProvider) Watts() (string, error) {
	uevents, err := f.UEvents()
	if err != nil {
		return "", err
	}
	current := uevents["POWER_SUPPLY_CURRENT_NOW"]
	voltage := uevents["POWER_SUPPLY_VOLTAGE_NOW"]

	var currentF float64
	if currentF, err = strconv.ParseFloat(current, 64); err != nil {
		return "", err
	}
	var voltageF float64
	if voltageF, err = strconv.ParseFloat(voltage, 64); err != nil {
		return "", err
	}

	watts := (currentF * voltageF) / 1000000000000
	return fmt.Sprint(watts), nil
}
