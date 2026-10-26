package attnsync

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/syncthing/syncthing/internal/db"
	"github.com/syncthing/syncthing/internal/db/sqlite"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/model"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/tlsutil"
	"github.com/thejerf/suture/v4"
)

type Engine struct {
	ID     protocol.DeviceID
	cfg    config.Wrapper
	model  model.Model
	sdb    db.DB
	keyGen *protocol.KeyGenerator
	cert   tls.Certificate
	sup    *suture.Supervisor
}

func New(cfg config.Wrapper, dbPath string, cert tls.Certificate, evLogger events.Logger) (*Engine, error) {
	sdb, err := sqlite.Open(dbPath)
	if err != nil {
		return nil, err
	}
	id := protocol.NewDeviceID(cert.Certificate[0])
	keyGen := protocol.NewKeyGenerator()
	m := model.NewModel(cfg, id, sdb, nil, evLogger, keyGen)
	sup := suture.NewSimple("attnsync")
	sup.Add(m)
	return &Engine{ID: id, cfg: cfg, model: m, sdb: sdb, keyGen: keyGen, cert: cert, sup: sup}, nil
}

func (e *Engine) Serve(ctx context.Context) error {
	err := e.sup.Serve(ctx)
	e.sdb.Close()
	return err
}

func (e *Engine) SetIgnores(folder string, lines []string) error {
	return e.model.SetIgnores(folder, lines)
}

func (e *Engine) ServeConn(ctx context.Context, raw net.Conn, dialer bool) error {
	tc := tlsutil.SecureDefaultTLS13()
	tc.Certificates = []tls.Certificate{e.cert}
	tc.NextProtos = []string{"bep/1.0"}
	tc.ClientAuth = tls.RequireAnyClientCert
	tc.SessionTicketsDisabled = true
	tc.InsecureSkipVerify = true
	var c *tls.Conn
	if dialer {
		c = tls.Client(raw, tc)
	} else {
		c = tls.Server(raw, tc)
	}
	_ = c.SetDeadline(time.Now().Add(20 * time.Second))
	if err := c.HandshakeContext(ctx); err != nil {
		raw.Close()
		return err
	}
	certs := c.ConnectionState().PeerCertificates
	if len(certs) != 1 {
		c.Close()
		return fmt.Errorf("peer presented %d certificates, want 1", len(certs))
	}
	remoteID := protocol.NewDeviceID(certs[0].Raw)
	devCfg, ok := e.cfg.Device(remoteID)
	if !ok || remoteID == e.ID {
		c.Close()
		return fmt.Errorf("unknown peer %s", remoteID.Short())
	}
	hello := protocol.Hello{ClientName: "attn", ClientVersion: "v2.1.6", Timestamp: time.Now().UnixNano(), NumConnections: 1}
	if me, ok := e.cfg.Device(e.ID); ok {
		hello.DeviceName = me.Name
	}
	theirs, err := protocol.ExchangeHello(c, hello)
	if err != nil {
		c.Close()
		return err
	}
	_ = c.SetDeadline(time.Time{})
	if err := e.model.OnHello(remoteID, raw.RemoteAddr(), theirs); err != nil {
		c.Close()
		return err
	}
	info := connInfo{Conn: c, id: fmt.Sprintf("%x-%x", hello.Timestamp, theirs.Timestamp), at: time.Now()}
	pc := protocol.NewConnection(remoteID, c, c, c, e.model, info, devCfg.Compression.ToProtocol(), e.keyGen)
	e.model.AddConnection(pc, theirs)
	select {
	case <-pc.Closed():
		return errors.New("connection closed")
	case <-ctx.Done():
		pc.Close(ctx.Err())
		return ctx.Err()
	}
}

type connInfo struct {
	*tls.Conn
	id string
	at time.Time
}

func (c connInfo) Type() string             { return "attn-link" }
func (c connInfo) Transport() string        { return "attn-link" }
func (c connInfo) IsLocal() bool            { return false }
func (c connInfo) Priority() int            { return 0 }
func (c connInfo) String() string           { return "attn-link " + c.RemoteAddr().String() }
func (c connInfo) Crypto() string           { return "TLS1.3" }
func (c connInfo) EstablishedAt() time.Time { return c.at }
func (c connInfo) ConnectionID() string     { return c.id }
