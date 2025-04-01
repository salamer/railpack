package golang

import (
	"testing"

	testingUtils "github.com/salamer/railpack/core/testing"
	"github.com/stretchr/testify/require"
)

func TestGolang(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		detected   bool
		hasGoMod   bool
		goVersion  string
		cgoEnabled bool
	}{
		{
			name:      "go mod",
			path:      "../../../examples/go-mod",
			detected:  true,
			hasGoMod:  true,
			goVersion: "1.23",
		},
		{
			name:      "go cmd dirs",
			path:      "../../../examples/go-cmd-dirs",
			detected:  true,
			hasGoMod:  true,
			goVersion: "1.18",
		},
		{
			name:     "node",
			path:     "../../../examples/node-npm",
			detected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			provider := GoProvider{}
			detected, err := provider.Detect(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.detected, detected)

			if detected {
				err = provider.Initialize(ctx)
				require.NoError(t, err)

				err = provider.Plan(ctx)
				require.NoError(t, err)

				require.Equal(t, tt.hasGoMod, provider.isGoMod(ctx))
				require.Equal(t, tt.cgoEnabled, provider.hasCGOEnabled(ctx))

				if tt.goVersion != "" {
					goVersion := ctx.Resolver.Get("go")
					require.Equal(t, tt.goVersion, goVersion.Version)
				}
			}
		})
	}
}
