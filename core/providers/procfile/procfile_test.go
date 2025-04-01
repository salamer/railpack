package procfile

import (
	"testing"

	testingUtils "github.com/salamer/railpack/core/testing"
	"github.com/stretchr/testify/require"
)

func TestProcfile(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/python-uv")
	provider := ProcfileProvider{}

	_, err := provider.Plan(ctx)
	require.NoError(t, err)

	require.Equal(t, "gunicorn --bind 0.0.0.0:3333 main:app", ctx.Deploy.StartCmd)
}
