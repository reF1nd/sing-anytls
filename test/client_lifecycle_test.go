package test

import (
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	anytls "github.com/sagernet/sing-anytls"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"github.com/stretchr/testify/require"
)

type countingConn struct {
	net.Conn
	once   sync.Once
	closed *atomic.Int32
}

func (c *countingConn) Close() error {
	c.once.Do(func() { c.closed.Add(1) })
	return c.Conn.Close()
}

func TestDisableReuseAndIdlePolicy(t *testing.T) {
	t.Parallel()
	for _, disableReuse := range []bool{false, true} {
		for _, keepIdle := range []bool{false, true} {
			for _, keepOnce := range []bool{false, true} {
				t.Run(strings.Join([]string{strconv.FormatBool(disableReuse), strconv.FormatBool(keepIdle), strconv.FormatBool(keepOnce)}, "/"), func(t *testing.T) {
					t.Parallel()
					server := startService(t, newService(t, nil, nil))
					echo := startTCPEcho(t)
					var dialed, closed atomic.Int32
					client, err := anytls.NewClient(anytls.ClientOptions{
						Password: testPassword, DisableReuse: disableReuse,
						DialOut: func(ctx context.Context) (net.Conn, error) {
							conn, err := (&net.Dialer{}).DialContext(ctx, N.NetworkTCP, server.String())
							if err != nil {
								return nil, err
							}
							dialed.Add(1)
							return &countingConn{Conn: conn, closed: &closed}, nil
						},
					})
					require.NoError(t, err)
					defer client.Close()
					client.SetKeepIdleConnections(keepIdle)
					dial := client.DialContext
					if keepOnce {
						dial = func(ctx context.Context, destination M.Socksaddr) (net.Conn, error) {
							return client.DialContext(anytls.ContextWithKeepSession(ctx), destination)
						}
					}
					for range 3 {
						echoOnce(t, dial, echo, 4096, false)
					}
					if disableReuse || !keepIdle && !keepOnce {
						require.EqualValues(t, 3, dialed.Load())
						require.EqualValues(t, 3, closed.Load())
					} else {
						require.EqualValues(t, 1, dialed.Load())
						require.Zero(t, closed.Load())
					}
					client.CloseIdleConnections()
					require.Equal(t, dialed.Load(), closed.Load())
				})
			}
		}
	}
}

func TestResetActiveAndIdleSessions(t *testing.T) {
	t.Parallel()
	server := startService(t, newService(t, nil, nil))
	echo := startTCPEcho(t)
	var closed atomic.Int32
	client, err := anytls.NewClient(anytls.ClientOptions{
		Password: testPassword,
		DialOut: func(ctx context.Context) (net.Conn, error) {
			conn, err := (&net.Dialer{}).DialContext(ctx, N.NetworkTCP, server.String())
			if err != nil {
				return nil, err
			}
			return &countingConn{Conn: conn, closed: &closed}, nil
		},
	})
	require.NoError(t, err)
	defer client.Close()
	active, err := client.DialContext(context.Background(), echo)
	require.NoError(t, err)
	defer active.Close()
	// Keep the first stream open while a second session becomes idle.
	echoOnce(t, client.DialContext, echo, 4096, false)
	client.CloseIdleConnections()
	require.EqualValues(t, 1, closed.Load())
	require.NoError(t, active.SetDeadline(time.Now().Add(5*time.Second)))
	_, err = active.Write([]byte("alive"))
	require.NoError(t, err)
	response := make([]byte, 5)
	_, err = io.ReadFull(active, response)
	require.NoError(t, err)
	require.Equal(t, "alive", string(response))
	echoOnce(t, client.DialContext, echo, 4096, false)
	client.Reset()
	require.EqualValues(t, 3, closed.Load())
	_, err = active.Write([]byte("closed"))
	require.Error(t, err)
	// Reset is not Close: subsequent TCP sessions must still work.
	echoOnce(t, client.DialContext, echo, 4096, false)
	require.NoError(t, client.Close())
	require.EqualValues(t, 4, closed.Load())
}
