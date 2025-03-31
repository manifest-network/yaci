package collectors

import (
	"database/sql"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

// CollectorFactory is a function type that creates a collector with provided parameters
type CollectorFactory func(db *sql.DB, extraParams ...interface{}) (prometheus.Collector, error)

type Registry struct {
	factories []CollectorFactory
}

func NewRegistry() *Registry {
	return &Registry{
		factories: make([]CollectorFactory, 0),
	}
}

func (r *Registry) Register(factory CollectorFactory) {
	r.factories = append(r.factories, factory)
}

// CreateCollectors instantiates all collectors using the provided parameters
func (r *Registry) CreateCollectors(db *sql.DB, extraParams ...interface{}) ([]prometheus.Collector, error) {
	if db == nil {
		return nil, errors.New("database connection is nil")
	}

	collectors := make([]prometheus.Collector, 0, len(r.factories))
	for _, factory := range r.factories {
		collector, err := factory(db, extraParams...)
		if err != nil {
			return nil, err
		}
		collectors = append(collectors, collector)
	}
	return collectors, nil
}

var DefaultRegistry = NewRegistry()

func RegisterCollectorFactory(factory CollectorFactory) {
	DefaultRegistry.Register(factory)
}
