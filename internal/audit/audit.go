// Package audit предоставляет механизмы для асинхронного сбора и отправки событий.
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
)

// Observer определяет контракт для обработчиков событий аудита.
type Observer interface {
	// Handle обрабатывает переданное событие.
	Handle(event Event) error
}

// Auditor координирует подписанных наблюдателей.
type Auditor struct {
	logger    *zap.Logger
	events    chan Event
	observers []Observer
	wg        sync.WaitGroup
	mu        sync.Mutex
}

// URLObserver реализует Observer для доставки событий на удалённый HTTP-эндпоинт.
type URLObserver struct {
	client *http.Client
	url    string
}

// FileObserver реализует Observer для записи событий в локальный файл.
type FileObserver struct {
	file *os.File
	path string
	mu   sync.Mutex
}

// Event предоставляет событие аудита, содержащее метаданные о произошедшем действии.
type Event struct {
	IPAddr  string   `json:"ip_address"`
	Metrics []string `json:"metrics"`
	TS      int64    `json:"ts"`
}

// Notify рассылает событие наблюдателям.
func (a *Auditor) Notify(e Event) {
	a.events <- e
}

func (a *Auditor) processEvent(ctx context.Context) {
	defer a.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-a.events:
			if !ok {
				return
			}
			for _, obs := range a.observers {
				if err := obs.Handle(e); err != nil {
					a.logger.Error("audit observer failed", zap.Error(err))
				}
			}
		}
	}

}

// Shutdown завершает оставшиеся уведомления и закрывает соединение.
func (a *Auditor) Shutdown() {
	close(a.events)
	for e := range a.events {
		for _, obs := range a.observers {
			if err := obs.Handle(e); err != nil {
				a.logger.Error("audit observer failed", zap.Error(err))
			}
		}
	}
	a.wg.Wait()
}

// Handle записывает событие в файл в формате JSON.
func (fo *FileObserver) Handle(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	fo.mu.Lock()
	defer fo.mu.Unlock()

	_, err = fo.file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// Handle отправляет событие по HTTP методом POST.
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

// NewAuditor создает новый экземпляр Auditor.
func NewAuditor(ctx context.Context, logger *zap.Logger) *Auditor {
	a := &Auditor{
		observers: []Observer{},
		logger:    logger,
		events:    make(chan Event, 1024),
		wg:        sync.WaitGroup{},
		mu:        sync.Mutex{},
	}
	a.wg.Add(1)
	go a.processEvent(ctx)
	return a
}

// Subscribe добавляет нового наблюдателя в список рассылки.
func (a *Auditor) Subscribe(o Observer) {
	a.mu.Lock()
	a.observers = append(a.observers, o)
	a.mu.Unlock()
}

// NewFileObserver создает новый FileObserver.
func NewFileObserver(path string) (*FileObserver, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &FileObserver{
		path: path,
		file: file,
		mu:   sync.Mutex{},
	}, nil
}

// NewURLObserver создает новый URLObserver.
func NewURLObserver(url string, client *http.Client) *URLObserver {
	return &URLObserver{
		url:    url,
		client: client,
	}
}
