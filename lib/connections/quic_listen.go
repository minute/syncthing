// Copyright (C) 2019 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"crypto/tls"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	quic "github.com/lucas-clemente/quic-go"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/nat"
)

func init() {
	factory := &quicListenerFactory{}
	for _, scheme := range []string{"quic", "quic4", "quic6"} {
		listeners[scheme] = factory
	}
}

type quicListener struct {
	onAddressesChangedNotifier

	uri     *url.URL
	cfg     *config.Wrapper
	tlsCfg  *tls.Config
	stop    chan struct{}
	conns   chan internalConn
	factory listenerFactory

	err error
	mut sync.RWMutex
}

func (t *quicListener) Serve() {
	t.mut.Lock()
	t.err = nil
	t.mut.Unlock()

	netw := strings.Replace(strings.ToLower(t.uri.Scheme), "quic", "udp", 1)
	tcaddr, err := net.ResolveUDPAddr(netw, t.uri.Host)
	if err != nil {
		t.mut.Lock()
		t.err = err
		t.mut.Unlock()
		l.Infoln("Listen (BEP/QUIC):", err)
		return
	}

	listener, err := net.ListenUDP(netw, tcaddr)
	if err != nil {
		t.mut.Lock()
		t.err = err
		t.mut.Unlock()
		l.Infoln("Listen (BEP/QUIC):", err)
		return
	}
	defer listener.Close()

	qlst, err := quic.Listen(listener, t.tlsCfg, nil)
	if err != nil {
		t.mut.Lock()
		t.err = err
		t.mut.Unlock()
		l.Infoln("Listen (BEP/QUIC):", err)
		return
	}

	l.Infof("QUIC listener (%v) starting", listener.LocalAddr())
	defer l.Infof("QUIC listener (%v) shutting down", listener.LocalAddr())

	acceptFailures := 0
	const maxAcceptFailures = 10

	for {
		sess, err := qlst.Accept()
		select {
		case <-t.stop:
			if err == nil {
				sess.Close()
			}
			return
		default:
		}
		if err != nil {
			if err, ok := err.(*net.OpError); !ok || !err.Timeout() {
				l.Warnln("Listen (BEP/QUIC): Accepting connection:", err)

				acceptFailures++
				if acceptFailures > maxAcceptFailures {
					// Return to restart the listener, because something
					// seems permanently damaged.
					return
				}

				// Slightly increased delay for each failure.
				time.Sleep(time.Duration(acceptFailures) * time.Second)
			}
			continue
		}

		acceptFailures = 0
		l.Debugln("Listen (BEP/QUIC): connect from", sess.RemoteAddr())

		strm, err := sess.AcceptStream()
		if err != nil {
			l.Warnln("Listen (BEP/QUIC): Accepting connection:", err)
			sess.Close()
			continue
		}

		t.conns <- internalConn{quicConnection{sess, strm}, connTypeQUICServer, quicPriority}
	}
}

func (t *quicListener) Stop() {
	close(t.stop)
}

func (t *quicListener) URI() *url.URL {
	return t.uri
}

func (t *quicListener) WANAddresses() []*url.URL {
	return []*url.URL{t.uri}
}

func (t *quicListener) LANAddresses() []*url.URL {
	return []*url.URL{t.uri}
}

func (t *quicListener) Error() error {
	t.mut.RLock()
	err := t.err
	t.mut.RUnlock()
	return err
}

func (t *quicListener) String() string {
	return t.uri.String()
}

func (t *quicListener) Factory() listenerFactory {
	return t.factory
}

func (t *quicListener) NATType() string {
	return "unknown"
}

type quicListenerFactory struct{}

func (f *quicListenerFactory) New(uri *url.URL, cfg *config.Wrapper, tlsCfg *tls.Config, conns chan internalConn, natService *nat.Service) genericListener {
	return &quicListener{
		uri:     fixupPort(uri, config.DefaultTCPPort),
		cfg:     cfg,
		tlsCfg:  tlsCfg,
		conns:   conns,
		stop:    make(chan struct{}),
		factory: f,
	}
}

func (quicListenerFactory) Valid(_ config.Configuration) error {
	// Always valid
	return nil
}
