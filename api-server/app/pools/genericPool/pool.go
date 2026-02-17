package genericPool

import (
	"log"
	"sync"
	"time"
)

const debugTag = "genericPool."

// PoolItem is used to store data in the pool.
type PoolItem struct {
	Data any
}

// PoolList is a map that holds the Pool items, keyed by a token (string).
type PoolList map[string]PoolItem

// Pool is a struct that holds the pool of Pool items.
type Pool struct {
	Pool PoolList
	mu   sync.RWMutex
}

func New() *Pool {
	return &Pool{
		Pool: make(PoolList),
	}
}

func (p *Pool) Add(token string, data PoolItem, attrib ...time.Duration) {
	p.mu.Lock()
	p.Pool[token] = data
	p.mu.Unlock()

	if len(attrib) > 0 {
		if attrib[0] > 0 {
			// Start a goroutine to remove the pool item after a timeout
			go p.ItemTimeOut(token, attrib[0]) //Remove the pool item after a timeout
		}
	}
}

func (p *Pool) Delete(token string) {
	p.mu.Lock()
	delete(p.Pool, token)
	p.mu.Unlock()
}

func (p *Pool) Get(token string) (PoolItem, bool) {
	p.mu.RLock()
	item, exists := p.Pool[token]
	p.mu.RUnlock()
	return item, exists
}

func (p *Pool) List() {
	p.mu.RLock()
	for i, v := range p.Pool {
		log.Printf("Pool item=%v, details=%+v", i, v)
	}
	p.mu.RUnlock()
}

//ctx := context.WithValue(context.Background(), token, server)
//ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
//go h.ItemTimeOut(token) //Remove the pool item after a timeout

// ItemTimeOut this is used to cancel the Auth process.
// Could use a context.WithCancel here ??????? to be invesitgated later. ?????????????
func (p *Pool) ItemTimeOut(token string, timeout time.Duration) {
	time.Sleep(timeout)
	p.mu.Lock()
	if _, ok := p.Pool[token]; ok {
		delete(p.Pool, token)
		log.Printf(debugTag+"Handler.ItemTimeOut()1 ****** Auth timed out: Pool server deleted ********, token=%s, timeout=%v", token, timeout)
	}
	p.mu.Unlock()
}
