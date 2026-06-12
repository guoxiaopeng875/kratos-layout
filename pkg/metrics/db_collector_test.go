package metrics_test

import (
	"database/sql"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/go-kratos/kratos-layout/pkg/metrics"
)

func TestDBCollector_Describe(t *testing.T) {
	db := &sql.DB{} // zero-value DB is sufficient for describe
	collector := metrics.NewDBCollector(db)

	ch := make(chan *prometheus.Desc, 10)
	collector.Describe(ch)
	close(ch)

	var descs []*prometheus.Desc
	for d := range ch {
		descs = append(descs, d)
	}

	require.Len(t, descs, 5, "should describe 5 DB metrics")
}

func TestDBCollector_Collect(t *testing.T) {
	db := &sql.DB{} // zero-value DB returns zeroed Stats
	collector := metrics.NewDBCollector(db)

	ch := make(chan prometheus.Metric, 10)
	collector.Collect(ch)
	close(ch)

	var collected []prometheus.Metric
	for m := range ch {
		collected = append(collected, m)
	}

	require.Len(t, collected, 5, "should collect 5 DB metrics")
}
