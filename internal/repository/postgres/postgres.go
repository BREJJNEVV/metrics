package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/BREJJNEVV/metrics/internal/model"
	"github.com/BREJJNEVV/metrics/internal/retry"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
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
	ctx := context.Background()
	err := retry.Do(ctx, isRetriablePG, func() error {
		_, err := p.db.ExecContext(ctx, queryInsertCounter, name, value)
		return err
	})
	return err
}

func (p *PsgsRepository) Set(name string, value float64) error {
	ctx := context.Background()
	err := retry.Do(ctx, isRetriablePG, func() error {
		_, err := p.db.ExecContext(context.Background(),
			queryInsertGauge,
			name, value,
		)
		return err
	})
	return err
}

func (p *PsgsRepository) Counters() map[string]int64 {
	newMap := make(map[string]int64)
	var rows *sql.Rows
	ctx := context.Background()

	err := retry.Do(ctx, isRetriablePG, func() error {
		var queryErr error
		rows, queryErr = p.db.QueryContext(ctx, `SELECT name, value FROM counter_metrics`)
		if queryErr != nil {
			if rows != nil {
				rows.Close()
			}
			return queryErr
		}
		return nil
	})
	if rows != nil {
		defer rows.Close()
	}

	if err != nil {
		p.logger.Error("failed to query counters", zap.Error(err))
		if rows != nil {
			rows.Err()
		}
		return newMap
	}
	if rows == nil {
		p.logger.Error("rows is nil after query counters")
		return newMap
	}
	var name string
	var value int64
	for rows.Next() {
		if scanErr := rows.Scan(&name, &value); scanErr != nil {
			p.logger.Error("failed to scan counter row", zap.Error(scanErr))
			break
		}
		newMap[name] = value
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		p.logger.Error("rows iteration error", zap.Error(rowsErr))
	}
	return newMap
}

func (p *PsgsRepository) Gauges() map[string]float64 {
	newMap := make(map[string]float64)
	var rows *sql.Rows
	ctx := context.Background()

	err := retry.Do(ctx, isRetriablePG, func() error {
		var queryErr error
		rows, queryErr = p.db.QueryContext(ctx, `SELECT name, value FROM gauge_metrics`)
		if queryErr != nil {
			if rows != nil {
				_ = rows.Close()
			}
			return queryErr
		}
		return nil
	})
	if rows != nil {
		defer rows.Close()
	}

	if err != nil {
		p.logger.Error("failed to query gauges", zap.Error(err))
		if rows != nil {
			rows.Err()
		}
		return newMap
	}
	if rows == nil {
		p.logger.Error("rows is nil after query gauges")
		return newMap
	}

	var name string
	var value float64
	for rows.Next() {
		if scanErr := rows.Scan(&name, &value); scanErr != nil {
			p.logger.Error("failed to scan gauge row", zap.Error(scanErr))
			break
		}
		newMap[name] = value
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		p.logger.Error("rows iteration error", zap.Error(rowsErr))
	}
	return newMap
}

func (p *PsgsRepository) GetCounter(name string) (int64, bool) {
	var value int64
	ctx := context.Background()
	err := retry.Do(ctx, isRetriablePG, func() error {
		row := p.db.QueryRowContext(ctx, `SELECT value FROM counter_metrics WHERE name = $1`, name)
		return row.Scan(&value)
	})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			p.logger.Error("failed to get counter", zap.String("name", name), zap.Error(err))
		}
		return 0, false
	}
	return value, true
}

func (p *PsgsRepository) GetGauge(name string) (float64, bool) {
	var value float64
	ctx := context.Background()
	err := retry.Do(ctx, isRetriablePG, func() error {
		row := p.db.QueryRowContext(ctx, `SELECT value FROM gauge_metrics WHERE name = $1`, name)
		return row.Scan(&value)
	})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			p.logger.Error("failed to get gauge", zap.String("name", name), zap.Error(err))
		}
		return 0, false
	}
	return value, true
}

func (p *PsgsRepository) UpdateBatch(mr []model.Metrics) error {
	ctx := context.Background()

	return retry.Do(ctx, isRetriablePG, func() error {
		tx, err := p.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, m := range mr {
			switch m.MType {
			case model.Gauge:
				if m.Value == nil {
					return errors.New("gauge value is nil")
				}
				_, err = tx.ExecContext(ctx, queryInsertGauge, m.ID, *m.Value)
				if err != nil {
					return err
				}
			case model.Counter:
				if m.Delta == nil {
					return errors.New("counter delta is nil")
				}
				_, err = tx.ExecContext(ctx, queryInsertCounter, m.ID, *m.Delta)
				if err != nil {
					return err
				}
			default:
				return errors.New("unknown metric type")
			}
		}
		return tx.Commit()
	})
}

func New(db *sql.DB, logger *zap.Logger) *PsgsRepository {
	return &PsgsRepository{db: db, logger: logger}
}

func isRetriablePG(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgerrcode.IsConnectionException(pgErr.Code)
	}
	return false
}
