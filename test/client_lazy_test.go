package test

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	anytls "github.com/sagernet/sing-anytls"
	N "github.com/sagernet/sing/common/network"

	"github.com/stretchr/testify/require"
)

// Model a TFO dialer's unestablished connection: addresses are unavailable until
// the first write. Neither session creation nor address wrapping may use them early.
type lazyAddressConn struct {
	net.Conn
	written atomic.Bool
}

func (c *lazyAddressConn) Write(p []byte) (int, error) { c.written.Store(true); return c.Conn.Write(p) }

func (c *lazyAddressConn) LocalAddr() net.Addr {
	if !c.written.Load() {
		return nil
	}
	return c.Conn.LocalAddr()
}

func (c *lazyAddressConn) RemoteAddr() net.Addr {
	if !c.written.Load() {
		return nil
	}
	return c.Conn.RemoteAddr()
}

func TestClientWithLazyConnection(t *testing.T) {
	t.Parallel()
	server := startService(t, newService(t, nil, nil))
	client, err := anytls.NewClient(anytls.ClientOptions{
		Password: testPassword,
		DialOut: func(ctx context.Context) (net.Conn, error) {
			conn, err := (&net.Dialer{}).DialContext(ctx, N.NetworkTCP, server.String())
			if err != nil {
				return nil, err
			}
			return &lazyAddressConn{Conn: conn}, nil
		},
	})
	require.NoError(t, err)
	defer client.Close()
	echoOnce(t, client.DialContext, startTCPEcho(t), 128*1024, true)
}
