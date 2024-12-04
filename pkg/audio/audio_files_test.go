package audio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoogleSay(t *testing.T) {
	err := GoogleSay("Hello World!")
	require.NoError(t, err, "failed to say hello world")
}
