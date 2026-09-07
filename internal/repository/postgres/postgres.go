package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/BREJJNEVV/metrics/internal/model"
	"go.uber.org/zap"
)

const queryInsertGauge string = `INSERT INTO gauge_metrics (name, value)
     VALUES ($1, $2)
     ON CONFLICT (name) DO UPDATE 
	 SET value = EXCLUDED.value`

const queryInsertCounter string = `INSERT INTO counter_metrics (name, value)
     VALUES ($1, $2)
     ON CONFLICT (name) DO UPDATE 
	 SET value = counter_metrics.value + EXCLUDED.value`

type PsgsRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

func (p *PsgsRepository) Add(name string, value int64) error {
	_, err := p.db.ExecContext(context.Background(),
		queryInsertCounter,
		name, value,
	)
	if err != nil {
		return err
	}
	return nil
}

func (p *PsgsRepository) Set(name string, value float64) error {
	_, err := p.db.ExecContext(context.Background(),
		queryInsertGauge,
		name, value,
	)
	if err != nil {
		return err
	}
	return nil
}

func (p *PsgsRepository) Counters() map[string]int64 {
	newMap := make(map[string]int64)
	rows, err := p.db.QueryContext(context.Background(),
		`SELECT name, value FROM counter_metrics`,
	)
	if err != nil {
		p.logger.Error("failed to query counters", zap.Error(err))
		return newMap
	}
	defer rows.Close()
	var name string
	var value int64
	for rows.Next() {
		err = rows.Scan(&name, &value)
		if err != nil {
			p.logger.Error("failed to scan vales", zap.Error(err))
			continue
		}
		newMap[name] = value
	}

	if rows.Err() != nil {
		p.logger.Error("rows iteration error")

	}

	return newMap
}

func (p *PsgsRepository) Gauges() map[string]float64 {
	newMap := make(map[string]float64)
	rows, err := p.db.QueryContext(context.Background(),
		`SELECT name, value FROM gauge_metrics`,
	)
	if err != nil {
		p.logger.Error("failed to query counters", zap.Error(err))
		return newMap
	}
	defer rows.Close()
	var name string
	var value float64
	for rows.Next() {
		err = rows.Scan(&name, &value)
		if err != nil {
			p.logger.Error("failed to scan vales", zap.Error(err))
			continue
		}
		newMap[name] = value
	}

	if rows.Err() != nil {
		p.logger.Error("rows iteration error")

	}

	return newMap
}

func (p *PsgsRepository) GetCounter(name string) (int64, bool) {

	row := p.db.QueryRowContext(context.Background(),
		`SELECT value FROM counter_metrics 
		WHERE name = $1`, name)
	var value int64
	err := row.Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false
		}
		p.logger.Error("failed to scan counter values", zap.Error(err))
		return 0, false
	}

	return value, true
}

func (p *PsgsRepository) GetGauge(name string) (float64, bool) {
	row := p.db.QueryRowContext(context.Background(),
		`SELECT value FROM gauge_metrics WHERE name = $1`, name)
	var value float64
	err := row.Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false
		}
		p.logger.Error("failed to scan gauge values", zap.Error(err))
		return 0, false
	}

	return value, true
}

func (p *PsgsRepository) UpdateBatch(mr []model.Metrics) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, request := range mr {
		switch request.MType {
		case model.Gauge:
			_, err := tx.ExecContext(context.Background(),
				queryInsertGauge,
				request.ID, *request.Value,
			)
			if err != nil {
				return err
			}
		case model.Counter:
			_, err := tx.ExecContext(context.Background(),
				queryInsertCounter,
				request.ID, *request.Delta,
			)
			if err != nil {
				return err
			}
		default:
			return errors.New("unknown metric type")
		}
	}
	return tx.Commit()
}

func New(db *sql.DB, logger *zap.Logger) *PsgsRepository {
	return &PsgsRepository{db: db, logger: logger}
}
