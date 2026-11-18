// Package audit предоставляет механизмы для асинхронного сбора и отправки событий.
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

// Observer определяет контракт для обработчиков событий аудита.
type Observer interface {
	// Handle обрабатывает переданное событие.
	Handle(event Event) error
}

// Auditor координирует подписанных наблюдателей.
type Auditor struct {
	observers []Observer
	logger    *zap.Logger
}

// URLObserver реализует Observer для доставки событий на удалённый HTTP-эндпоинт.
type URLObserver struct {
	url    string
	client *http.Client
}

// FileObserver реализует Observer для записи событий в локальный файл.
type FileObserver struct {
	path string
	mu   sync.Mutex
}

// Event предоставляет событие аудита, содержащее метаданные о произошедшем действии.
type Event struct {
	// TS - время создания события.
	TS int64 `json:"ts"`

	// Metrics - слайс имён метрик.
	Metrics []string `json:"metrics"`

	// IPAddr - адрес отправки события.
	IPAddr string `json:"ip_address"`
}

// Notify рассылает событие наблюдателям.
func (a *Auditor) Notify(e Event) {
	for _, obs := range a.observers {
		if err := obs.Handle(e); err != nil {
			a.logger.Error("audit observer failed", zap.Error(err))
		}
	}
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
func NewAuditor(logger *zap.Logger) *Auditor {
	return &Auditor{
		observers: []Observer{},
		logger:    logger,
	}
}

// Subscribe добавляет нового наблюдателя в список рассылки.
func (a *Auditor) Subscribe(o Observer) {
	a.observers = append(a.observers, o)
}

// NewFileObserver создает новый FileObserver.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{
		path: path,
		mu:   sync.Mutex{},
	}
}

// NewURLObserver создает новый URLObserver.
func NewURLObserver(url string, client *http.Client) *URLObserver {
	return &URLObserver{
		url:    url,
		client: client,
	}
}
