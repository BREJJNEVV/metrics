package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"go.uber.org/zap"
)

type Observer interface {
	Handle(event Event) error
}

type Auditor struct {
	observers []Observer
	logger    *zap.Logger
}

type URLObserver struct {
	url    string
	client *http.Client
}

type FileObserver struct {
	path string
	mu   sync.Mutex
}

type Event struct {
	TS      int64    `json:"ts"`
	Metrics []string `json:"metrics"`
	IPAddr  string   `json:"ip_address"`
}

func (a *Auditor) Notify(e Event) {
	for _, obs := range a.observers {
		if err := obs.Handle(e); err != nil {
			a.logger.Error("audit observer failed", zap.Error(err))
		}
	}
}

func (fo *FileObserver) Handle(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	fo.mu.Lock()
	defer fo.mu.Unlock()
	file, err := os.OpenFile(fo.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (uo *URLObserver) Handle(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, uo.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := uo.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("audit url returned status %d", resp.StatusCode)
	}
	return nil
}

func NewAuditor(logger *zap.Logger) *Auditor {
	return &Auditor{
		observers: []Observer{},
		logger:    logger,
	}
}

func (s *Auditor) Subscribe(o Observer) {
	s.observers = append(s.observers, o)
}

func NewFileObserver(path string) *FileObserver {
	return &FileObserver{
		path: path,
		mu:   sync.Mutex{},
	}
}

func NewURLObserver(url string, client *http.Client) *URLObserver {
	return &URLObserver{
		url:    url,
		client: client,
	}
}
