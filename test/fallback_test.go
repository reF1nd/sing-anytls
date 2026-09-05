package test

import (
	"bytes"
	"context"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	anytls "github.com/sagernet/sing-anytls"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"github.com/stretchr/testify/require"
)

type fallbackCapture struct {
	payload []byte
	result  chan []byte
}

func (h fallbackCapture) NewConnectionEx(ctx context.Context, conn net.Conn, source M.Socksaddr, destination M.Socksaddr, onClose N.CloseHandlerFunc) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	data := make([]byte, len(h.payload))
	n, _ := io.ReadFull(conn, data)
	h.result <- data[:n]
	if onClose != nil {
		onClose(nil)
	}
}

func TestFallbackPreservesRequest(t *testing.T) {
	t.Parallel()
	for _, multi := range []bool{false, true} {
		for _, payload := range [][]byte{[]byte("GET /\r\n"), bytes.Repeat([]byte{0x11}, 128)} {
			t.Run(strconv.FormatBool(multi)+"/"+strconv.Itoa(len(payload)), func(t *testing.T) {
				t.Parallel()
				result := make(chan []byte, 1)
				fallback := fallbackCapture{payload, result}
				var service connectionService
				if multi {
					created, err := anytls.NewMultiService[string](anytls.ServiceOptions{Handler: proxyHandler{}, FallbackHandler: fallback})
					require.NoError(t, err)
					require.NoError(t, created.UpdateUsers([]string{"user"}, []string{testPassword}))
					service = created
				} else {
					service = newService(t, nil, fallback)
				}
				server := startService(t, service)
				conn, err := net.Dial(N.NetworkTCP, server.String())
				require.NoError(t, err)
				defer conn.Close()
				_, err = conn.Write(payload)
				require.NoError(t, err)
				select {
				case received := <-result:
					require.Equal(t, payload, received)
				case <-time.After(5 * time.Second):
					t.Fatal("fallback did not receive request")
				}
			})
		}
	}
}
