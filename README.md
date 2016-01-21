# SignalFX client for [go-metrics](https://github.com/rcrowley/go-metrics)

This is a simple client for go-metrics that submits all stats to SignalFX's HTTP API.

Usage:

```
	conf := signalfx.Config{
		Token: "API token",
		Prefix: "go",
		Dimensions: map[string]string{
			"app": "gopherbook",
			"host": "gb-prod-1",
			"env": "production",
		},
	}
	go signalfx.SignalFX(metrics.DefaultRegistry, 5 * time.Second, conf)
```
