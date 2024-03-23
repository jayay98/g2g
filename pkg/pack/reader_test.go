package pack

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPackReader(t *testing.T) {

	inputs := []string{
		"0076ca82a6dff817ec66f44342007202690a93763949 15027957951b64cf874c3557a0f3547bd83b3ff6 refs/heads/master report-status\n",
		"006c0000000000000000000000000000000000000000 cdfdb42577e2506715f8cfeacdbabc092bf63e8d refs/heads/experiment\n",
		"00000009done\n",
	}
	br := NewReader(strings.NewReader(strings.Join(inputs, "")))

	out := make([]byte, 128)
	n, err := br.Read(out)
	outFit := make([]byte, n)
	copy(outFit, out)

	require.Equal(t, string(outFit), inputs[0])
	require.NoError(t, err)
}
