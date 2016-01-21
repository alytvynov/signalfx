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

const DefaultAddr = "https://ingest.signalfx.com/v2/datapoint"

type Config struct {
	Addr   string
	Token  string
	Prefix string
}

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
	vals := buildBody(r, config.Prefix)
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
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
}

func buildBody(r metrics.Registry, prefix string) map[string]metric {
	vals := make(map[string]metric)

	r.Each(func(name string, i interface{}) {
		name = prefix + "." + name
		switch m := i.(type) {
		case metrics.Counter:
			vals["counter"] = metric{Metric: name, Value: float64(m.Count())}
		case metrics.Gauge:
			vals["gauge"] = metric{Metric: name, Value: float64(m.Value())}
		case metrics.GaugeFloat64:
			vals["gauge"] = metric{Metric: name, Value: m.Value()}
		case metrics.Histogram:
			vals["gauge"] = metric{Metric: name, Value: m.Mean()}
		case metrics.Meter:
			vals["gauge"] = metric{Metric: name, Value: m.Rate1()}
		case metrics.Timer:
			vals["gauge"] = metric{Metric: name, Value: m.Mean()}
		}
	})

	return vals
}
