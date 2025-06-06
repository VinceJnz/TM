package handlerAuth

import (
	"log"

	"github.com/1Password/srp"
)

type srpPoolItem struct {
	serverSRP *srp.SRP
	userID    int
}

type srpPoolList map[string]srpPoolItem

func (h *Handler) PoolAdd(token string, userID int, srpServer *srp.SRP) {
	p := srpPoolItem{
		serverSRP: srpServer,
		userID:    userID,
	}
	h.Pool[token] = p
}

func (h *Handler) PoolDelete(token string) {
	delete(h.Pool, token)
}

func (h *Handler) PoolGet(token string) srpPoolItem {
	return h.Pool[token]
}

func (h *Handler) PoolList() {
	for i, v := range h.Pool {
		log.Printf("Pool item=%v, details=%+v", i, v)
	}
}
