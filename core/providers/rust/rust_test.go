package rust

import (
	"testing"

	testingUtils "github.com/salamer/railpack/core/testing"
	"github.com/stretchr/testify/require"
)

func TestRust(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		detected    bool
		rustVersion string
	}{
		{
			name:        "rust ring",
			path:        "../../../examples/rust-ring",
			detected:    true,
			rustVersion: "1.84.0",
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
			provider := RustProvider{}
			detected, err := provider.Detect(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.detected, detected)

			if detected {
				err = provider.Initialize(ctx)
				require.NoError(t, err)

				err = provider.Plan(ctx)
				require.NoError(t, err)

				if tt.rustVersion != "" {
					rustVersion := ctx.Resolver.Get("rust")
					require.Equal(t, tt.rustVersion, rustVersion.Version)
				}
			}
		})
	}
}
