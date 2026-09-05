package test

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	anytls "github.com/sagernet/sing-anytls"
	M "github.com/sagernet/sing/common/metadata"

	"github.com/stretchr/testify/require"
)

// Check the wire, including the client= key when its value is explicitly empty.
func TestClientMetadataWire(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name     string
		metadata *string
		expected string
	}{
		{name: "default", expected: anytls.DefaultClientMetadata},
		{name: "empty", metadata: stringPointer("")},
		{name: "custom", metadata: stringPointer("sing-anytls/0.0.13 sing-box/test"), expected: "sing-anytls/0.0.13 sing-box/test"},
		{name: "equals", metadata: stringPointer("custom=value"), expected: "custom=value"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			local, peer := net.Pipe()
			defer peer.Close()
			require.NoError(t, peer.SetDeadline(time.Now().Add(5*time.Second)))
			settings := make(chan string, 1)
			go func() {
				var request [34]byte
				if _, err := io.ReadFull(peer, request[:]); err != nil {
					return
				}
				if _, err := io.CopyN(io.Discard, peer, int64(binary.BigEndian.Uint16(request[32:]))); err != nil {
					return
				}
				for {
					var header [7]byte
					if _, err := io.ReadFull(peer, header[:]); err != nil {
						return
					}
					payload := make([]byte, binary.BigEndian.Uint16(header[5:]))
					if _, err := io.ReadFull(peer, payload); err != nil {
						return
					}
					if header[0] == 4 {
						settings <- string(payload)
					}
				}
			}()
			client, err := anytls.NewClient(anytls.ClientOptions{
				Password: testPassword, ClientMetadata: testCase.metadata,
				DialOut: func(context.Context) (net.Conn, error) { return local, nil },
			})
			require.NoError(t, err)
			defer client.Close()
			conn, err := client.DialContext(context.Background(), M.ParseSocksaddr("example.com:443"))
			require.NoError(t, err)
			require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
			_, err = conn.Write(nil)
			require.NoError(t, err)
			select {
			case data := <-settings:
				require.Contains(t, strings.Split(data, "\n"), "client="+testCase.expected)
				require.Contains(t, strings.Split(data, "\n"), "v=2")
			case <-time.After(5 * time.Second):
				t.Fatal("no settings frame")
			}
		})
	}
}

func stringPointer(value string) *string { return &value }
