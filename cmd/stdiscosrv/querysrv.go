// Copyright (C) 2014-2015 Jakob Borg and Contributors (see the CONTRIBUTORS file).

package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"path"

	"github.com/golang/groupcache/lru"
	"golang.org/x/time/rate"
)

type querysrv struct {
	addr     string
	limiter  *safeCache
	listener net.Listener
	addrs    agingMap
}

type safeCache struct {
	*lru.Cache
	mut sync.Mutex
}

func (s *safeCache) Get(key string) (val interface{}, ok bool) {
	s.mut.Lock()
	val, ok = s.Cache.Get(key)
	s.mut.Unlock()
	return
}

func (s *safeCache) Add(key string, val interface{}) {
	s.mut.Lock()
	s.Cache.Add(key, val)
	s.mut.Unlock()
}

func (s *querysrv) Serve() {
	s.limiter = &safeCache{
		Cache: lru.New(lruSize),
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Println("Listen:", err)
		return
	}
	s.listener = listener

	http.HandleFunc("/v3/", s.handler)
	http.HandleFunc("/ping", handlePing)

	srv := &http.Server{
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		MaxHeaderBytes: 1 << 10,
	}

	if err := srv.Serve(s.listener); err != nil {
		log.Println("Serve:", err)
	}
}

func (s *querysrv) handler(w http.ResponseWriter, req *http.Request) {
	var remoteIP net.IP
	if remote := req.Header.Get("X-Forwarded-For"); remote != "" {
		remoteIP = net.ParseIP(remote)
	} else {
		addr, err := net.ResolveTCPAddr("tcp", req.RemoteAddr)
		if err != nil {
			log.Println("remoteAddr:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		remoteIP = addr.IP
	}

	if s.limit(remoteIP) {
		w.Header().Set("Retry-After", "300")
		http.Error(w, "Too Many Requests", 429)
		return
	}

	switch req.Method {
	case "GET":
		s.handleGET(w, req)
	case "POST":
		s.handlePOST(remoteIP, w, req)
	default:
		globalStats.Error()
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (s *querysrv) handleGET(w http.ResponseWriter, req *http.Request) {
	globalStats.Query()
	token := req.URL.Query().Get("token")

	addrs, ok := s.addrs.get(token)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	globalStats.Answer()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(addrs)
}

func (s *querysrv) handlePOST(remoteIP net.IP, w http.ResponseWriter, req *http.Request) {
	var addrs []string
	if err := json.NewDecoder(req.Body).Decode(&addrs); err != nil {
		globalStats.Error()
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	token := path.Base(req.URL.Path)
	s.addrs.add(token, addrs)

	globalStats.Announce()

	w.Header().Set("Reannounce-After", "3600")
	w.WriteHeader(http.StatusNoContent)
}

func (s *querysrv) Stop() {
	s.listener.Close()
}

func (s *querysrv) limit(remote net.IP) bool {
	key := remote.String()

	bkt, ok := s.limiter.Get(key)
	if ok {
		bkt := bkt.(*rate.Limiter)
		if !bkt.Allow() {
			// Rate limit exceeded; ignore request
			return true
		}
	} else {
		// limitAvg is in requests per ten seconds.
		s.limiter.Add(key, rate.NewLimiter(rate.Limit(limitAvg)/10, limitBurst))
	}

	return false
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
