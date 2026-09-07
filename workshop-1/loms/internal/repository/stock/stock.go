package stock

import (
	"sync"

	"route256/loms/internal/model"
)

type entry struct {
	total    uint64
	reserved uint64
}

type Repository struct {
	mu   sync.Mutex
	data map[uint32]*entry
}

func New(seed map[uint32]uint64) *Repository {
	data := make(map[uint32]*entry, len(seed))
	for sku, total := range seed {
		data[sku] = &entry{total: total}
	}
	return &Repository{data: data}
}

func (r *Repository) GetAvailable(sku uint32) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.data[sku]
	if !ok {
		return 0
	}
	return e.total - e.reserved
}

func (r *Repository) Reserve(sku uint32, count uint16) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.data[sku]
	if !ok || e.total-e.reserved < uint64(count) {
		return model.ErrInsufficient
	}
	e.reserved += uint64(count)
	return nil
}

func (r *Repository) Release(sku uint32, count uint16) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.data[sku]
	if !ok || e.reserved < uint64(count) {
		return model.ErrInsufficient
	}
	e.reserved -= uint64(count)
	return nil
}

func (r *Repository) RemoveReserved(sku uint32, count uint16) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.data[sku]
	if !ok || e.reserved < uint64(count) || e.total < uint64(count) {
		return model.ErrInsufficient
	}
	e.reserved -= uint64(count)
	e.total -= uint64(count)
	return nil
}
