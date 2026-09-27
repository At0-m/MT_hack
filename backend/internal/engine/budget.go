package engine

import (
	"context"
	"sync"
	"time"
	d "tramflow/internal/domain"
)

type requestBudget struct {
	mu    sync.Mutex
	used  int
	users map[string]int
}

// 32 units globally, 30 per user; one unit covers at most 256 hourly cells.
// A full 7440-cell request takes 30 units, so only one fits at a time.
func (b *requestBudget) acquire(ctx context.Context, q d.Selection) (func(), error) {
	cells := len(q.RouteIDs) * int(q.Window.To.Sub(q.Window.From)/time.Hour)
	units := (cells + 255) / 256
	user := d.UserID(ctx)
	if user == "" {
		user = "internal"
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.users == nil {
		b.users = map[string]int{}
	}
	if b.used+units > 32 || b.users[user]+units > 30 {
		return nil, d.Fail(429, "CALCULATION_BUSY", "Лимит объёма параллельных расчётов; повторите запрос.")
	}
	b.used += units
	b.users[user] += units
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.used -= units
		b.users[user] -= units
		if b.users[user] == 0 {
			delete(b.users, user)
		}
	}, nil
}
