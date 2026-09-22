package memory

import (
	"context"
	"testing"

	"github.com/BREJJNEVV/metrics/internal/model"
)

type gaugeCase struct {
	testname string
	name     string
	value    float64
}

func TestSetGauge(t *testing.T) {
	data := []gaugeCase{
		{testname: "simple", name: "Alloc", value: 123.45},
		{testname: "zero", name: "Zero", value: 0},
	}

	for _, c := range data {
		t.Run(c.testname, func(t *testing.T) {
			mem := Create()
			err := mem.Set(context.Background(), c.name, c.value)
			if err != nil {
				t.Fatalf("Set failed: %v", err)
			}
			got, ok := mem.GetGauge(context.Background(), c.name)
			if !ok {
				t.Fatalf("gauge %q not found", c.name)
			}
			if got != c.value {
				t.Errorf("expected %v, got %v", c.value, got)
			}
		})
	}
}

type counterCase struct {
	testname string
	name     string
	value    int64
}

func TestAddCounter(t *testing.T) {
	data := []counterCase{
		{testname: "simple", name: "PollCount", value: 12},
		{testname: "zero", name: "Zero", value: 0},
	}

	for _, c := range data {
		t.Run(c.testname, func(t *testing.T) {
			mem := Create()
			err := mem.Add(context.Background(), c.name, c.value)
			if err != nil {
				t.Fatalf("Add failed: %v", err)
			}
			got, ok := mem.GetCounter(context.Background(), c.name)
			if !ok {
				t.Fatalf("gauge %q not found", c.name)
			}
			if got != c.value {
				t.Errorf("expected %v, got %v", c.value, got)
			}
		})
	}
}

func TestAddCounterAccumulates(t *testing.T) {
	mem := Create()
	ctx := context.Background()
	_ = mem.Add(ctx, "PollCount", 5)
	_ = mem.Add(ctx, "PollCount", 3)
	_ = mem.Add(ctx, "PollCount", 2)

	got, ok := mem.GetCounter(ctx, "PollCount")
	if !ok {
		t.Fatal("counter not found")
	}
	if got != 10 {
		t.Errorf("expected 10, got %d", got)
	}
}

func TestSetGaugeReplaces(t *testing.T) {
	mem := Create()
	ctx := context.Background()
	_ = mem.Set(ctx, "Alloc", 100)
	_ = mem.Set(ctx, "Alloc", 999)

	got, _ := mem.GetGauge(ctx, "Alloc")
	if got != 999 {
		t.Errorf("expected 999, got %v", got)
	}
}
func TestGetGaugeNotFoun(t *testing.T) {
	mem := Create()
	got, ok := mem.GetGauge(context.Background(), "unknown")
	if ok {
		t.Error("expected ok=false for missing gauge")
	}
	if got != 0 {
		t.Errorf("expected 0, got %v", got)
	}
}

func TestGetCounterNotFound(t *testing.T) {
	mem := Create()
	got, ok := mem.GetCounter(context.Background(), "unknown")
	if ok {
		t.Error("expected ok=false for missing counter")
	}
	if got != 0 {
		t.Errorf("expected 0, got %v", got)
	}
}

func TestGauges(t *testing.T) {
	mem := Create()
	ctx := context.Background()
	gotMap := mem.Gauges(ctx)
	if len(gotMap) != 0 {
		t.Errorf("expected empty map, got %d entries", len(gotMap))
	}

	_ = mem.Set(ctx, "Alloc", 123.45)
	_ = mem.Set(ctx, "Sys", 1000)

	g := mem.Gauges(ctx)
	if g["Alloc"] != 123.45 {
		t.Errorf("expected 123.45, got %v", g["Alloc"])
	}
	if g["Sys"] != 1000 {
		t.Errorf("expected 1000, got %v", g["Sys"])
	}
	if len(g) != 2 {
		t.Errorf("expected 2 entries, got %d", len(g))
	}

	gotMap["Hacked"] = 9999
	gotMap["Alloc"] = 0

	gotMap2 := mem.Gauges(ctx)
	if _, ok := gotMap2["Hacked"]; ok {
		t.Error("external mutation leaked into storage")
	}
	if gotMap2["Alloc"] != 123.45 {
		t.Errorf("external mutation changed stored value: %v", gotMap2["Alloc"])
	}
}

func TestCounters(t *testing.T) {
	mem := Create()
	ctx := context.Background()

	gotMap := mem.Counters(ctx)
	if len(gotMap) != 0 {
		t.Errorf("expected empty map, got %d entries", len(gotMap))
	}

	_ = mem.Add(ctx, "PollCount", 12)
	_ = mem.Add(ctx, "Sys", 1000)

	c := mem.Counters(ctx)
	if c["PollCount"] != 12 {
		t.Errorf("expected 12, got %v", c["PollCount"])
	}
	if c["Sys"] != 1000 {
		t.Errorf("expected 1000, got %v", c["Sys"])
	}
	if len(c) != 2 {
		t.Errorf("expected 2 entries, got %d", len(c))
	}

	gotMap["Hacked"] = 9999
	gotMap["PollCount"] = 0

	gotMap2 := mem.Counters(ctx)
	if _, ok := gotMap2["Hacked"]; ok {
		t.Error("external mutation leaked into storage")
	}
	if gotMap2["PollCount"] != 12 {
		t.Errorf("external mutation changed stored value: %v", gotMap2["PollCount"])
	}
}

func ptr(v float64) *float64 { return &v }
func ptrInt(v int64) *int64  { return &v }

func TestUpdateBatch(t *testing.T) {
	mem := Create()
	ctx := context.Background()

	batch := []model.Metrics{
		{ID: "Alloc", MType: model.Gauge, Value: ptr(123.45)},
		{ID: "PollCount", MType: model.Counter, Delta: ptrInt(10)},
	}

	if err := mem.UpdateBatch(ctx, batch); err != nil {
		t.Fatalf("UpdateBatch failed: %v", err)
	}

	g, ok := mem.GetGauge(ctx, "Alloc")
	if !ok || g != 123.45 {
		t.Errorf("expected gauge 123.45, got %v (ok=%v)", g, ok)
	}

	c, ok := mem.GetCounter(ctx, "PollCount")
	if !ok || c != 10 {
		t.Errorf("expected counter 10, got %v (ok=%v)", c, ok)
	}
}
