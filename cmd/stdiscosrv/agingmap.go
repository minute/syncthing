package main

import (
	"log"
	"sync"
	"time"
)

type agingMap struct {
	cur     map[string][]string
	prev    map[string][]string
	curHour int64
	mut     sync.RWMutex
}

func (m *agingMap) get(token string) ([]string, bool) {
	m.mut.RLock()
	defer m.mut.RUnlock()
	if vals, ok := m.cur[token]; ok {
		return vals, true
	}
	if vals, ok := m.prev[token]; ok {
		return vals, true
	}
	return nil, false
}

func (m *agingMap) add(token string, vals []string) {
	m.mut.Lock()
	defer m.mut.Unlock()

	if now := time.Now().Unix() / 3600; now != m.curHour {
		// Time to rotate the maps
		log.Printf("Rotate: %d tokens in map", len(m.cur))
		m.prev = m.cur
		m.cur = make(map[string][]string, len(m.prev))
		m.curHour = now
	}

	exs, ok := m.cur[token]
	if !ok {
		// fast path
		m.cur[token] = vals
		return
	}

	// merge
nextVal:
	for _, val := range vals {
		for _, ex := range exs {
			if val == ex {
				continue nextVal
			}
		}
		exs = append(exs, val)
	}
	m.cur[token] = exs
}
