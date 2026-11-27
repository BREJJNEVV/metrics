// Package model описывает доменные сущности проекта: метрики, запросы и ответы.
package model

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.

// Metrics представляет собой объект метрики для передачи в JSON-запросах и ответах.
// generate:reset
type Metrics struct {
	// ID содержит имя метрики.
	ID    string `json:"id"`
	MType string `json:"type"`

	// Delta хранит значение приращения для метрик counter.
	Delta *int64 `json:"delta,omitempty"`

	// Value хранит текущее значение метрик gauge.
	Value *float64 `json:"value,omitempty"`

	// Hash содержит SHA256 подпись тела для проверки целостности.
	Hash string `json:"hash,omitempty"`
}
