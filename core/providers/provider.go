package providers

import (
	"github.com/salamer/railpack/core/generate"
	"github.com/salamer/railpack/core/providers/deno"
	"github.com/salamer/railpack/core/providers/golang"
	"github.com/salamer/railpack/core/providers/java"
	"github.com/salamer/railpack/core/providers/node"
	"github.com/salamer/railpack/core/providers/php"
	"github.com/salamer/railpack/core/providers/python"
	"github.com/salamer/railpack/core/providers/ruby"
	"github.com/salamer/railpack/core/providers/rust"
	"github.com/salamer/railpack/core/providers/shell"
	"github.com/salamer/railpack/core/providers/staticfile"
)

type Provider interface {
	Name() string
	Detect(ctx *generate.GenerateContext) (bool, error)
	Initialize(ctx *generate.GenerateContext) error
	Plan(ctx *generate.GenerateContext) error
	StartCommandHelp() string
}

func GetLanguageProviders() []Provider {
	// Order is important here. The first provider that returns true from Detect() will be used.
	return []Provider{
		&php.PhpProvider{},
		&golang.GoProvider{},
		&java.JavaProvider{},
		&python.PythonProvider{},
		&rust.RustProvider{},
		&ruby.RubyProvider{},
		&deno.DenoProvider{},
		&node.NodeProvider{},
		&staticfile.StaticfileProvider{},
		&shell.ShellProvider{},
	}
}

func GetProvider(name string) Provider {
	for _, provider := range GetLanguageProviders() {
		if provider.Name() == name {
			return provider
		}
	}

	return nil
}
