package srpPool

import (
	"log"
	"time"

	"github.com/1Password/srp"
)

const debugTag = "srpPool."

// PoolItem is used to store the SRP server and user ID in the pool.
type PoolItem struct {
	ServerSRP *srp.SRP
	UserID    int
}

// PoolList is a map that holds the SRP pool items, keyed by a token (string).
type PoolList map[string]PoolItem

// srpPool is a struct that holds the pool of SRP items.
type Pool struct {
	Pool PoolList
}

func New() *Pool {
	return &Pool{
		Pool: make(PoolList),
	}
}

func (p *Pool) Add(token string, userID int, srpServer *srp.SRP, attrib ...time.Duration) {
	log.Printf(debugTag+"Handler.Add()1 - Adding SRP server to pool with token: %s, userID: %d, attrib: %+v", token, userID, attrib)
	i := PoolItem{
		ServerSRP: srpServer,
		UserID:    userID,
	}
	p.Pool[token] = i

	if len(attrib) > 0 {
		if attrib[0] > 0 {
			// Start a goroutine to remove the pool item after a timeout
			go p.ItemTimeOut(token, attrib[0]) //Remove the pool item after a timeout
		}
	}
}

func (p *Pool) Delete(token string) {
	delete(p.Pool, token)
}

func (p *Pool) Get(token string) PoolItem {
	return p.Pool[token]
}

func (p *Pool) List() {
	for i, v := range p.Pool {
		log.Printf("Pool item=%v, details=%+v", i, v)
	}
}

//ctx := context.WithValue(context.Background(), token, server)
//ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
//go h.ItemTimeOut(token) //Remove the pool item after a timeout

// ItemTimeOut this is used to cancel the Auth process.
// Could use a context.WithCancel here ??????? to be invesitgated later. ?????????????
func (p *Pool) ItemTimeOut(token string, timeout time.Duration) {
	time.Sleep(timeout)
	if _, ok := p.Pool[token]; ok {
		p.Delete(token)
		log.Printf(debugTag+"Handler.ItemTimeOut()1 ****** Auth timed out: Pool server deleted ********, token=%s, timeout=%v", token, timeout)
	}
}
