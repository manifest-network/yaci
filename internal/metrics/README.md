# Metrics Module

This module provides Prometheus metrics for monitoring the YACI application.

## Overview

The `metrics` module defines collectors that gather data from various sources (e.g., the database) and expose them as Prometheus metrics.
These metrics can then be scraped by a Prometheus server for monitoring and alerting purposes.

## Collectors

The following collectors are currently implemented:

-   **TotalTransactionCountCollector**: Collects the total number of transactions stored in the database.
-   **TotalUniqueAddressesCollector**: Collects the total number of unique user and group addresses stored in the database.

## Usage

To use the metrics module:

1.  Create a new collector instance, passing in any required dependencies (e.g., a database connection).
2.  Register the collector with a Prometheus registry.

## Example

```go
package collectors

import "github.com/prometheus/client_golang/prometheus"

// ExampleCollector is a Prometheus collector that collects example metrics.
type ExampleCollector struct {
    // Add any dependencies or data sources here, e.g., a database connection.
    exampleMetric *prometheus.Desc
}

// NewExampleCollector creates a new ExampleCollector.
func NewExampleCollector() *ExampleCollector {
    return &ExampleCollector{
        // Initialize any dependencies or data sources here.
        exampleMetric: prometheus.NewDesc(
            prometheus.BuildFQName("example", "metric", "name"),
            "Description of the example metric",
            nil,
            prometheus.Labels{"source": "example"},
        ),
    }
}

// Describe sends the super-set of all possible descriptors of metrics collected by this Collector
// to the provided channel and returns once the last descriptor has been sent.
func (c *ExampleCollector) Describe(ch chan<- *prometheus.Desc) {
    ch <- c.exampleMetric
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *ExampleCollector) Collect(ch chan<- prometheus.Metric) {
    // Collect the value of the example metric.
    // If there is an error, send an invalid metric.
    // Otherwise, send the metric value.
    ch <- prometheus.MustNewConstMetric(c.exampleMetric, prometheus.GaugeValue, float64(123))
}
```