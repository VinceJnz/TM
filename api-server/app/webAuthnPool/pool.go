package webAuthnPool

import (
	"api-server/v2/models"
	"log"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

const debugTag = "webAuthnPool."

// PoolItem is used to store the server and user ID in the pool.
type PoolItem struct {
	SessionData *webauthn.SessionData
	User        *models.User //webauthn.User
}

// PoolList is a map that holds the pool items, keyed by a token (string).
type PoolList map[string]PoolItem

// Pool is a struct that holds the pool of items.
type Pool struct {
	Pool PoolList
}

func NewSRPPool() *Pool {
	return &Pool{
		Pool: make(PoolList),
	}
}

func (p *Pool) Add(token string, user *models.User, sessionData *webauthn.SessionData, attrib ...time.Duration) {
	i := PoolItem{
		SessionData: sessionData,
		User:        user,
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
		log.Printf(debugTag + "Handler.ItemTimeOut()1 ****** Auth timed out: Pool server deleted ********")
	}
}
