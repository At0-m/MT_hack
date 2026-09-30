package storage

import (
	"context"
	"github.com/jackc/pgx/v5"
	d "tramflow/internal/domain"
)

func insertHours(ctx context.Context, tx pgx.Tx, p d.Provenance, hours []d.Hour) error {
	for start := 0; start < len(hours); start += 128 {
		batch := &pgx.Batch{}
		for _, h := range hours[start:min(start+128, len(hours))] {
			calendar := map[string]float64{}
			for _, key := range []string{"is_holiday", "is_weekend"} {
				if value, ok := h.Features[key]; ok {
					calendar[key] = value
				}
			}
			batch.Queue("INSERT INTO prepared_features VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING", p.History, p.Features, p.Origin, h.RouteID, h.Time, h.FeaturesAvailableAt, encode(h.Features), h.SyntheticBoardings, encode(calendar))
			batch.Queue("INSERT INTO supply_profiles VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING", p.Supply, h.RouteID, h.Time, h.Fleet, h.Source, h.Proxy)
			batch.Queue("INSERT INTO reference_profiles VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING", p.Reference, h.RouteID, h.Time, h.Reference, h.Typical)
			batch.Queue("INSERT INTO weather_points VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING", p.Weather, h.RouteID, h.Time, encode(h.Weather))
		}
		result := tx.SendBatch(ctx, batch)
		for range batch.Len() {
			if _, err := result.Exec(); err != nil {
				_ = result.Close()
				return err
			}
		}
		if err := result.Close(); err != nil {
			return err
		}
	}
	return nil
}
