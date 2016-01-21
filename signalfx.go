package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rcrowley/go-metrics"
)

// DefaultAddr is the default API gateway used if Config.Addr is empty.
const DefaultAddr = "https://ingest.signalfx.com/v2/datapoint"

type Config struct {
	// SignalFX API endpoint for datapoint ingestion. Can be left empty for
	// default.
	Addr string
	// SignalFX API token.
	Token string
	// Prefix added to all metric names. Optional.
	Prefix string
	// Dimensions is a set of attributes added to each metric. It usually
	// contains things such as hostname, app name, environment name.
	Dimensions map[string]string
}

// SignalFX flushes values from registry every d interval.
//
// Note that this is does not spawn a worker goroutine to do the flushing. This
// function blocks indefinitely. Normally you want to start it as a goroutine.
func SignalFX(r metrics.Registry, d time.Duration, config Config) {
	if config.Addr == "" {
		config.Addr = DefaultAddr
	}
	for range time.Tick(d) {
		if err := send(r, config); err != nil {
			log.Println("signalfx:", err)
		}
	}
}

func send(r metrics.Registry, config Config) error {
	vals := buildBody(r, config)
	if len(vals) == 0 {
		// nothing to send
		return nil
	}
	body, err := json.Marshal(vals)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", config.Addr, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-SF-Token", config.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response code when posting metrics was %q", resp.Status)
	}

	return nil
}

type metric struct {
	Metric     string            `json:"metric"`
	Value      float64           `json:"value"`
	Dimensions map[string]string `json:"dimensions"`
}

func buildBody(r metrics.Registry, config Config) map[string][]metric {
	vals := make(map[string][]metric)

	r.Each(func(name string, i interface{}) {
		if config.Prefix != "" {
			name = config.Prefix + "." + name
		}
		switch m := i.(type) {
		case metrics.Counter:
			vals["counter"] = append(vals["counter"],
				metric{Metric: name, Value: float64(m.Count())})
		case metrics.Gauge:
			vals["gauge"] = append(vals["gauge"],
				metric{Metric: name, Value: float64(m.Value())})
		case metrics.GaugeFloat64:
			vals["gauge"] = append(vals["gauge"],
				metric{Metric: name, Value: m.Value()})
		case metrics.Histogram:
			vals["gauge"] = append(vals["gauge"],
				metric{Metric: name, Value: m.Mean()})
		case metrics.Meter:
			vals["gauge"] = append(vals["gauge"],
				metric{Metric: name, Value: m.Rate1()})
		case metrics.Timer:
			vals["gauge"] = append(vals["gauge"],
				metric{Metric: name, Value: m.Mean()})
		}
	})

	// Add dimensions to each metric. Separate loop to unclutter the switch
	// above.
	for _, ms := range vals {
		for i := range ms {
			ms[i].Dimensions = config.Dimensions
		}
	}

	return vals
}
