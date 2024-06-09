package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Device struct {
	ID       int    `json:"id"`
	Mac      string `json:"mac"`
	Firmware string `json:"firmware"`
}
type metrics struct {
	devices  prometheus.Gauge
	info     *prometheus.GaugeVec
	upgrades *prometheus.CounterVec
}

func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		devices: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "myapp",
			Name:      "connected_devices",
			Help:      "Number of currently connected devices.",
		}),
		info: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "myapp",
			Name:      "info",
			Help:      "Information about the My App environment.",
		},
			[]string{"version"}),
		upgrades: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "myapp",
			Name:      "device_upgrade_total",
			Help:      "Number of upgraded devices.",
		}, []string{"type"}),
	}
	reg.MustRegister(m.devices, m.info, m.upgrades)
	return m
}

var devices []Device
var version string

func init() {
	devices = []Device{
		{1, "5F-33-CC-1F-43-82", "2.1.6"},
		{2, "EF-2B-C4-F5-D6-34", "2.1.6"},
	}

	version = "2.10.5"

}

func main() {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	m.devices.Set(float64(len(devices)))

	m.info.With(prometheus.Labels{"version": version}).Set(1) // num apps with this version

	dMux := http.NewServeMux()
	rdh := registerDevicesHandler{metrics: m}
	mdh := manageDevicesHandler{metrics: m}
	dMux.Handle("/devices", rdh)
	dMux.Handle("/devices/", mdh)

	pMux := http.NewServeMux()
	promHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	pMux.Handle("/metrics", promHandler)

	go func() {
		log.Println("devices listening on ", "8080")
		log.Fatal(http.ListenAndServe(":8080", dMux))
	}()
	go func() {
		log.Println("metrics listening on ", "8081")
		log.Fatal(http.ListenAndServe(":8081", pMux))
	}()

	select {}
}

type registerDevicesHandler struct {
	metrics *metrics
}

func (rdh registerDevicesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getDevices(w, r)
	case "POST":
		createDevice(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func createDevice(w http.ResponseWriter, r *http.Request) {
	var device Device

	err := json.NewDecoder(r.Body).Decode(&device)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	devices = append(devices, device)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("device created"))
}

func getDevices(res http.ResponseWriter, req *http.Request) {
	data, err := json.Marshal(devices)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadGateway)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(data)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func upgradeDevice(w http.ResponseWriter, r *http.Request, m *metrics) {
	path := strings.TrimPrefix(r.URL.Path, "/devices/")
	// get the id
	id, err := strconv.Atoi(path)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	var device Device
	err = json.NewDecoder(r.Body).Decode(&device)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for i := range devices {
		if devices[i].ID == id {
			devices[i].Firmware = device.Firmware
		}
	}

	m.upgrades.With(prometheus.Labels{"type": "router"}).Inc()
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Upgrading..."))

}

type manageDevicesHandler struct {
	metrics *metrics
}

func (mdh manageDevicesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		upgradeDevice(w, r, mdh.metrics)
	default:
		w.Header().Set("Allow", "PUT")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
