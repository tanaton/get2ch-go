package process

import (
	"sync"
	"time"
)

const (
	BOURBON_TIME    = 1 * time.Minute
	BOARD_NAME_TIME = 24 * time.Hour
)

type BoardServerBox struct {
	m   map[string]string
	mux sync.RWMutex
}

func NewBoardServerBox(f func() map[string]string) *BoardServerBox {
	bs := &BoardServerBox{
		m: f(),
	}
	go func(bs *BoardServerBox) {
		c := time.Tick(1 * time.Hour)
		for _ = range c {
			bs.mux.Lock()
			bs.m = f()
			bs.mux.Unlock()
		}
	}(bs)
	return bs
}
func (bs *BoardServerBox) GetServer(board string) (server string) {
	bs.mux.RLock()
	server = bs.m[board]
	bs.mux.RUnlock()
	return
}

type boardNamePacket struct {
	board string
	name  string
}
type BoardNameBox struct {
	m   map[string]string
	wch chan<- boardNamePacket
	mux sync.RWMutex
}

func NewBoardNameBox() *BoardNameBox {
	ch := make(chan boardNamePacket, 4)
	bn := &BoardNameBox{
		m:   make(map[string]string, 1024),
		wch: ch,
	}
	go func(bn *BoardNameBox, rch <-chan boardNamePacket) {
		c := time.Tick(BOARD_NAME_TIME)
		for {
			select {
			case it := <-rch:
				bn.mux.Lock()
				bn.m[it.board] = it.name
				bn.mux.Unlock()
			case <-c:
				bn.mux.Lock()
				bn.m = make(map[string]string, 1024)
				bn.mux.Unlock()
			}
		}
	}(bn, ch)
	return bn
}
func (bn *BoardNameBox) SetName(board, bname string) {
	bnp := boardNamePacket{
		board: board,
		name:  bname,
	}
	bn.wch <- bnp
}
func (bn *BoardNameBox) GetName(board string) (name string) {
	bn.mux.RLock()
	name = bn.m[board]
	bn.mux.RUnlock()
	return
}

type BBNCacheBox struct {
	cm  map[string]time.Time
	wch chan<- string
	dch chan<- string
	mux sync.RWMutex
}

func NewBBNCacheBox() *BBNCacheBox {
	ch := make(chan string, 1)
	dch := make(chan string, 1)
	bbn := &BBNCacheBox{
		cm:  make(map[string]time.Time),
		wch: ch,
		dch: dch,
	}
	go func(bbn *BBNCacheBox, rch <-chan string, drch <-chan string) {
		for {
			select {
			case key := <-rch:
				// バーボン期間設定
				bbn.mux.Lock()
				bbn.cm[key] = time.Now().Add(BOURBON_TIME)
				bbn.mux.Unlock()
			case key := <-drch:
				bbn.mux.Lock()
				delete(bbn.cm, key)
				bbn.mux.Unlock()
			}
		}
	}(bbn, ch, dch)
	return bbn
}
func (bbn *BBNCacheBox) SetBourbon(key string) {
	bbn.wch <- key
}
func (bbn *BBNCacheBox) GetBourbon(key string) bool {
	bbn.mux.RLock()
	t, ok := bbn.cm[key]
	bbn.mux.RUnlock()
	if ok {
		if time.Now().After(t) {
			// 期間経過
			bbn.dch <- key
			ok = false
		}
	}
	return ok
}
