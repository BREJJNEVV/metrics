package audit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestFileObserver_Handle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	obs := NewFileObserver(path)

	event := Event{
		TS:      12345678,
		Metrics: []string{"Alloc", "PollCount"},
		IPAddr:  "127.0.0.1",
	}

	if err := obs.Handle(event); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if !bytes.HasSuffix(data, []byte("\n")) {
		t.Error("file should end with newline")
	}

	var got Event
	if err := json.Unmarshal(bytes.TrimSuffix(data, []byte("\n")), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.TS != event.TS {
		t.Errorf("TS: got %d, want %d", got.TS, event.TS)
	}
	if len(got.Metrics) != 2 || got.Metrics[0] != "Alloc" || got.Metrics[1] != "PollCount" {
		t.Errorf("Metrics: got %v, want [Alloc PollCount]", got.Metrics)
	}
	if got.IPAddr != event.IPAddr {
		t.Errorf("IPAddr: got %q, want %q", got.IPAddr, event.IPAddr)
	}
}

func TestFileObserver_Appends(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	obs := NewFileObserver(path)

	e1 := Event{TS: 1, Metrics: []string{"A"}, IPAddr: "1.1.1.1"}
	e2 := Event{TS: 2, Metrics: []string{"B"}, IPAddr: "2.2.2.2"}

	if err := obs.Handle(e1); err != nil {
		t.Fatalf("Handle e1: %v", err)
	}
	if err := obs.Handle(e2); err != nil {
		t.Fatalf("Handle e2: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], `"metrics":["A"]`) {
		t.Errorf("first line: %s", lines[0])
	}
	if !strings.Contains(lines[1], `"metrics":["B"]`) {
		t.Errorf("second line: %s", lines[1])
	}
}

func TestURLObserver_Handle(t *testing.T) {
	var (
		gotMethod      string
		gotContentType string
		gotBody        []byte
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	obs := NewURLObserver(srv.URL, &http.Client{})
	event := Event{TS: 42, Metrics: []string{"X"}, IPAddr: "9.9.9.9"}

	if err := obs.Handle(event); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method: got %s, want POST", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type: got %q", gotContentType)
	}

	var parsed Event
	if err := json.Unmarshal(gotBody, &parsed); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if parsed.TS != event.TS || parsed.IPAddr != event.IPAddr {
		t.Errorf("event mismatch: %+v", parsed)
	}
}

func TestURLObserver_HandleNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	obs := NewURLObserver(srv.URL, &http.Client{})
	if err := obs.Handle(Event{TS: 1}); err == nil {
		t.Error("expected error on non-200 status")
	}
}

// ---------- Auditor.Notify ----------

type mockObserver struct {
	events []Event
	err    error
}

func (m *mockObserver) Handle(e Event) error {
	m.events = append(m.events, e)
	return m.err
}

func TestAuditor_Notify(t *testing.T) {
	logger := zap.NewNop()
	a := NewAuditor(logger)

	o1 := &mockObserver{}
	o2 := &mockObserver{}
	a.Subscribe(o1)
	a.Subscribe(o2)

	event := Event{TS: 7, Metrics: []string{"M"}, IPAddr: "0.0.0.0"}
	a.Notify(event)

	if len(o1.events) != 1 || o1.events[0].TS != 7 {
		t.Errorf("o1 didn't receive event: %+v", o1.events)
	}
	if len(o2.events) != 1 || o2.events[0].TS != 7 {
		t.Errorf("o2 didn't receive event: %+v", o2.events)
	}
}

func TestAuditor_NotifyContinuesOnError(t *testing.T) {
	logger := zap.NewNop()
	a := NewAuditor(logger)

	bad := &mockObserver{err: io.ErrUnexpectedEOF}
	good := &mockObserver{}
	a.Subscribe(bad)
	a.Subscribe(good)

	a.Notify(Event{TS: 1})

	if len(good.events) != 1 {
		t.Error("good observer should still receive event after bad one failed")
	}
}

func TestAuditor_NotifyNoObservers(t *testing.T) {
	logger := zap.NewNop()
	a := NewAuditor(logger)

	// не должно паниковать
	a.Notify(Event{TS: 1})
}
