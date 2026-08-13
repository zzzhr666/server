package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewRegistryRegistersStandardCollectors(t *testing.T) {
	registry := NewRegistry()

	metricFamilies, err := registry.Gatherer().Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	metricFamilyNames := make(map[string]struct{}, len(metricFamilies))
	for _, metricFamily := range metricFamilies {
		metricFamilyNames[metricFamily.GetName()] = struct{}{}
	}

	for _, name := range []string{"go_goroutines", "process_cpu_seconds_total"} {
		if _, ok := metricFamilyNames[name]; !ok {
			t.Errorf("Gather() missing metric family %q", name)
		}
	}
}

func TestRegistriesAreIsolated(t *testing.T) {
	firstRegistry := NewRegistry()
	secondRegistry := NewRegistry()
	firstCounter := prometheus.NewCounter(prometheus.CounterOpts{Name: "game_test_operations_total"})
	secondCounter := prometheus.NewCounter(prometheus.CounterOpts{Name: "game_test_operations_total"})

	if err := firstRegistry.Registerer().Register(firstCounter); err != nil {
		t.Fatalf("register first counter: %v", err)
	}
	if err := secondRegistry.Registerer().Register(secondCounter); err != nil {
		t.Fatalf("register second counter with same name: %v", err)
	}

	firstCounter.Add(1)
	secondCounter.Add(2)

	if got := counterValue(t, firstRegistry, "game_test_operations_total"); got != 1 {
		t.Errorf("first registry counter = %v, want 1", got)
	}
	if got := counterValue(t, secondRegistry, "game_test_operations_total"); got != 2 {
		t.Errorf("second registry counter = %v, want 2", got)
	}
}

func counterValue(t *testing.T, registry *Registry, name string) float64 {
	t.Helper()

	metricFamilies, err := registry.Gatherer().Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	for _, metricFamily := range metricFamilies {
		if metricFamily.GetName() != name {
			continue
		}
		if len(metricFamily.GetMetric()) != 1 {
			t.Fatalf("metric family %q has %d metrics, want 1", name, len(metricFamily.GetMetric()))
		}
		counter := metricFamily.GetMetric()[0].GetCounter()
		if counter == nil {
			t.Fatalf("metric family %q is not a counter", name)
		}
		return counter.GetValue()
	}

	t.Fatalf("metric family %q not found", name)
	return 0
}
