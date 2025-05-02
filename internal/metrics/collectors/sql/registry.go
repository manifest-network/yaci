package sql

import (
	"database/sql"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

// SqlCollectorFactory is a function type that creates a collector with provided parameters
type SqlCollectorFactory func(db *sql.DB, extraParams ...interface{}) (prometheus.Collector, error)

type SqlRegistry struct {
	factories []SqlCollectorFactory
}

func NewSqlRegistry() *SqlRegistry {
	return &SqlRegistry{
		factories: make([]SqlCollectorFactory, 0),
	}
}

func (r *SqlRegistry) Register(factory SqlCollectorFactory) {
	r.factories = append(r.factories, factory)
}

// CreateSqlCollectors instantiates all collectors using the provided parameters
func (r *SqlRegistry) CreateSqlCollectors(db *sql.DB, extraParams ...interface{}) ([]prometheus.Collector, error) {
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

var DefaultSqlRegistry = NewSqlRegistry()

func RegisterCollectorFactory(factory SqlCollectorFactory) {
	DefaultSqlRegistry.Register(factory)
}
