package db

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
	//logger
	//mutex
}

func Create() *MemStorage {
	return &MemStorage{
		counter: make(map[string]int64),
		gauge:   make(map[string]float64),
	}
}

// возможнл позже добавить дженерики
func (r *MemStorage) Set(name string, value float64) error {
	//Тип gauge, float64 — новое значение должно замещать предыдущее.
	r.gauge[name] = value

	return nil
}

func (r *MemStorage) Add(name string, value int64) error {
	//Тип counter, int64 — новое значение должно добавляться к предыдущему
	// - если какое-то значение уже было известно серверу.
	_, ok := r.counter[name]
	if ok {
		r.counter[name] += value
	} else {
		r.counter[name] = value
	}

	return nil
}

func (r *MemStorage) GetGauge(name string) (float64, bool) {
	v, ok := r.gauge[name]
	if !ok {
		return 0, false
	}
	return v, true
}

func (r *MemStorage) GetCounter(name string) (int64, bool) {
	v, ok := r.counter[name]
	if !ok {
		return 0, false
	}
	return v, true
}

func (ms *MemStorage) Gauges() map[string]float64 {
	return ms.gauge
}

func (ms *MemStorage) Counters() map[string]int64 {
	return ms.counter
}
