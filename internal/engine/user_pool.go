package engine

import (
	"context"
	"log"
	"sync"
	"time"

	"honda/go-engine/internal/db"
)

type userPool struct {
	users    []db.User
	store    *db.Store
	cooldown time.Duration

	mu    sync.Mutex
	index int
}

func newUserPool(users []db.User, store *db.Store, cooldown time.Duration) *userPool {
	return &userPool{
		users:    users,
		store:    store,
		cooldown: cooldown,
	}
}

func (p *userPool) Acquire(ctx context.Context) (*db.User, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		now := time.Now()

		p.mu.Lock()
		n := len(p.users)
		var selected *db.User

		for i := 0; i < n; i++ {
			idx := (p.index + i) % n
			u := &p.users[idx]

			if u.Token == "" {
				continue
			}
			if u.InFlight.Valid && u.InFlight.Int64 > 0 {
				continue
			}
			if u.CooldownUntil.Valid && u.CooldownUntil.Time.After(now) {
				continue
			}
			if u.BlockedUntil.Valid && u.BlockedUntil.Time.After(now) {
				continue
			}

			p.index = idx + 1
			selected = u
			break
		}
		p.mu.Unlock()

		if selected != nil {
			cooldownUntil := now.Add(p.cooldown)
			if err := p.store.ReserveUser(ctx, selected.ID, now, cooldownUntil); err != nil {
				log.Printf("[go] reserve user=%s err=%v", selected.CodUsuario, err)
			}
			p.mu.Lock()
			selected.InFlight.Int64 = 1
			selected.InFlight.Valid = true
			selected.LastRequest.Time = now
			selected.LastRequest.Valid = true
			selected.CooldownUntil.Time = cooldownUntil
			selected.CooldownUntil.Valid = true
			p.mu.Unlock()
			return selected, nil
		}

		time.Sleep(5 * time.Millisecond)
	}
}

func (p *userPool) Release(ctx context.Context, user *db.User) {
	_ = p.store.ReleaseUser(ctx, user.ID)
	p.mu.Lock()
	user.InFlight.Int64 = 0
	user.InFlight.Valid = true
	p.mu.Unlock()
}
