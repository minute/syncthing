// Copyright (C) 2019 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package connections

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"time"

	quic "github.com/lucas-clemente/quic-go"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

func init() {
	factory := &quicDialerFactory{}
	for _, scheme := range []string{"quic", "quic4", "quic6"} {
		dialers[scheme] = factory
	}
}

type quicDialer struct {
	cfg    *config.Wrapper
	tlsCfg *tls.Config
}

func (d *quicDialer) Dial(id protocol.DeviceID, uri *url.URL) (internalConn, error) {
	uri = fixupPort(uri, config.DefaultTCPPort)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sess, err := quic.DialAddrContext(ctx, uri.Host, d.tlsCfg, nil)
	if err != nil {
		return internalConn{}, err
	}

	strm, err := sess.OpenStream()
	if err != nil {
		return internalConn{}, err
	}

	return internalConn{quicConnection{sess, strm}, connTypeQUICClient, quicPriority}, nil
}

func (d *quicDialer) RedialFrequency() time.Duration {
	return time.Duration(d.cfg.Options().ReconnectIntervalS) * time.Second
}

type quicDialerFactory struct{}

func (quicDialerFactory) New(cfg *config.Wrapper, tlsCfg *tls.Config) genericDialer {
	return &tcpDialer{
		cfg:    cfg,
		tlsCfg: tlsCfg,
	}
}

func (quicDialerFactory) Priority() int {
	return quicPriority
}

func (quicDialerFactory) AlwaysWAN() bool {
	return false
}

func (quicDialerFactory) Valid(_ config.Configuration) error {
	// Always valid
	return nil
}

func (quicDialerFactory) String() string {
	return "QUIC Dialer"
}

type quicConnection struct {
	quic.Session
	quic.Stream
}

func (c quicConnection) Close() error {
	if err := c.Stream.Close(); err != nil {
		return err
	}
	return c.Session.Close()
}

func (c quicConnection) PeerCertificates() []*x509.Certificate {
	cs := c.Session.ConnectionState()
	return cs.PeerCertificates
}
