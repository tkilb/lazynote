package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand(t *testing.T) {
	tests := []struct {
		goos string
		name string
		ok   bool
	}{
		{"darwin", "open", true},
		{"linux", "xdg-open", true},
		{"freebsd", "xdg-open", true},
		{"windows", "", false},
		{"plan9", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			name, args, err := command(tt.goos, "https://example.com/a?b=1")
			if !tt.ok {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.name, name)
			assert.Equal(t, []string{"https://example.com/a?b=1"}, args, "the address is one argument, never a shell string")
		})
	}
}

func TestCommandRefusesAnythingButWebAddresses(t *testing.T) {
	for _, raw := range []string{"-a Calculator", "file:///etc/passwd", "javascript:alert(1)", "", "https://x.com/ y", "ftp://x.com"} {
		_, _, err := command("darwin", raw)
		assert.Error(t, err, raw)
	}
}
