package core

import "time"

type Reconciler struct {
	store    Store
	backends []Backend
	interval time.Duration
}

func NewReconciler(store Store, backends []Backend, interval time.Duration) *Reconciler {
	return &Reconciler{store, backends, interval}
}

func (r *Reconciler) Start() {
	go func() {
		for {
			time.Sleep(r.interval)
			// future: reconciliation logic
		}
	}()
}
