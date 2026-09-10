package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/BREJJNEVV/metrics/internal/model"
	"github.com/BREJJNEVV/metrics/internal/retry"
	"github.com/BREJJNEVV/metrics/migrations"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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
	db     *pgxpool.Pool
	logger *zap.Logger
}

func (p *PsgsRepository) Add(ctx context.Context, name string, value int64) error {
	err := retry.Do(ctx, isRetriablePG, func() error {
		_, err := p.db.Exec(ctx, queryInsertCounter, name, value)
		return err
	})
	return err
}

func (p *PsgsRepository) Set(ctx context.Context, name string, value float64) error {
	err := retry.Do(ctx, isRetriablePG, func() error {
		_, err := p.db.Exec(ctx,
			queryInsertGauge,
			name, value,
		)
		return err
	})
	return err
}

func (p *PsgsRepository) Counters(ctx context.Context) map[string]int64 {
	newMap := make(map[string]int64)
	var rows pgx.Rows
	var err error

	for attempt := range retry.Attempts {
		rows, err = p.db.Query(ctx, `SELECT name, value FROM counter_metrics`)
		if err == nil {
			break
		}
		if !isRetriablePG(err) || attempt == retry.Attempts-1 {
			p.logger.Error("failed to query counters", zap.Error(err))
			return newMap
		}
		select {
		case <-ctx.Done():
			return newMap
		case <-time.After(time.Duration(1+(retry.Delay*attempt)) * time.Second):
		}
	}

	if rows == nil {
		p.logger.Error("rows is nil after query counters")
		return newMap
	}
	defer rows.Close()

	var name string

	var value int64
	for rows.Next() {
		if err := rows.Scan(&name, &value); err != nil {
			p.logger.Error("failed to scan counter row", zap.Error(err))
			continue
		}
		newMap[name] = value
	}

	if err := rows.Err(); err != nil {
		p.logger.Error("rows iteration error", zap.Error(err))
	}

	return newMap
}

func (p *PsgsRepository) Gauges(ctx context.Context) map[string]float64 {
	newMap := make(map[string]float64)
	var rows pgx.Rows
	var err error

	for attempt := range retry.Attempts {
		rows, err = p.db.Query(ctx, `SELECT name, value FROM gauge_metrics`)
		if err == nil {
			break
		}
		if !isRetriablePG(err) || attempt == retry.Attempts-1 {
			p.logger.Error("failed to query gauges", zap.Error(err))
			return newMap
		}
		select {
		case <-ctx.Done():
			return newMap
		case <-time.After(time.Duration(1+(retry.Delay*attempt)) * time.Second):
		}
	}

	if rows == nil {
		p.logger.Error("rows is nil after query gauges")
		return newMap
	}
	defer rows.Close()

	var name string
	var value float64
	for rows.Next() {
		if err := rows.Scan(&name, &value); err != nil {
			p.logger.Error("failed to scan gauge row", zap.Error(err))
			continue
		}
		newMap[name] = value
	}

	if err := rows.Err(); err != nil {
		p.logger.Error("rows iteration error", zap.Error(err))
	}

	return newMap
}

func (p *PsgsRepository) GetCounter(ctx context.Context, name string) (int64, bool) {
	var value int64
	err := retry.Do(ctx, isRetriablePG, func() error {
		row := p.db.QueryRow(ctx, `SELECT value FROM counter_metrics WHERE name = $1`, name)
		return row.Scan(&value)
	})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			p.logger.Error("failed to get counter", zap.String("name", name), zap.Error(err))
		}
		return 0, false
	}
	return value, true
}

func (p *PsgsRepository) GetGauge(ctx context.Context, name string) (float64, bool) {
	var value float64
	err := retry.Do(ctx, isRetriablePG, func() error {
		row := p.db.QueryRow(ctx, `SELECT value FROM gauge_metrics WHERE name = $1`, name)
		return row.Scan(&value)
	})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			p.logger.Error("failed to get gauge", zap.String("name", name), zap.Error(err))
		}
		return 0, false
	}
	return value, true
}

func (p *PsgsRepository) UpdateBatch(ctx context.Context, mr []model.Metrics) error {
	return retry.Do(ctx, isRetriablePG, func() error {
		tx, err := p.db.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		for _, m := range mr {
			switch m.MType {
			case model.Gauge:
				if m.Value == nil {
					return errors.New("gauge value is nil")
				}
				_, err = tx.Exec(ctx, queryInsertGauge, m.ID, *m.Value)
				if err != nil {
					return err
				}
			case model.Counter:
				if m.Delta == nil {
					return errors.New("counter delta is nil")
				}
				_, err = tx.Exec(ctx, queryInsertCounter, m.ID, *m.Delta)
				if err != nil {
					return err
				}
			default:
				return errors.New("unknown metric type")
			}
		}
		return tx.Commit(ctx)
	})
}

func New(db *pgxpool.Pool, logger *zap.Logger, dsn string) (*PsgsRepository, error) {
	err := runMigrations(dsn)
	if err != nil {
		return nil, err
	}
	return &PsgsRepository{db: db, logger: logger}, err
}

func runMigrations(dsn string) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return err
	}
	defer m.Close()
	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func isRetriablePG(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgerrcode.IsConnectionException(pgErr.Code) ||
			pgerrcode.IsTransactionRollback(pgErr.Code)
	}
	return false
}

func (p *PsgsRepository) Ping(ctx context.Context) error {
	start := time.Now()
	if err := p.db.Ping(ctx); err != nil {
		return err
	}
	p.logger.Info("db ping ok", zap.Duration("duration", time.Since(start)))
	return nil
}
