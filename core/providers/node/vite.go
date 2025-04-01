package node

import (
	"regexp"
	"strings"

	"github.com/salamer/railpack/core/generate"
)

const (
	DefaultViteOutputDirectory = "dist"
)

func (p *NodeProvider) isVite(ctx *generate.GenerateContext) bool {
	hasViteConfig := ctx.App.HasMatch("vite.config.js") || ctx.App.HasMatch("vite.config.ts")
	hasViteBuildCommand := strings.Contains(strings.ToLower(p.packageJson.GetScript("build")), "vite build")

	// SvelteKit does not build as a static site by default
	if p.isSvelteKit() {
		return false
	}

	return hasViteConfig || hasViteBuildCommand
}

func (p *NodeProvider) getViteOutputDirectory(ctx *generate.GenerateContext) string {
	configContent := ""

	if ctx.App.HasMatch("vite.config.js") {
		content, err := ctx.App.ReadFile("vite.config.js")
		if err == nil {
			configContent = content
		}
	} else if ctx.App.HasMatch("vite.config.ts") {
		content, err := ctx.App.ReadFile("vite.config.ts")
		if err == nil {
			configContent = content
		}
	}

	if configContent != "" {
		// Look for outDir in config
		outDirRegex := regexp.MustCompile(`outDir:\s*['"](.+?)['"]`)
		matches := outDirRegex.FindStringSubmatch(configContent)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	// Check for outDir in build script
	if p.packageJson.Scripts != nil {
		if buildScript, ok := p.packageJson.Scripts["build"]; ok {
			outDirRegex := regexp.MustCompile(`vite\s+build(?:\s+-[^\s]*)*\s+(?:--outDir)\s+([^-\s;]+)`)
			matches := outDirRegex.FindStringSubmatch(buildScript)
			if len(matches) > 1 {
				return matches[1]
			}
		}
	}

	return DefaultViteOutputDirectory
}

func (p *NodeProvider) getViteCache(ctx *generate.GenerateContext) string {
	return ctx.Caches.AddCache("vite", "node_modules/.vite")
}

func (p *NodeProvider) isSvelteKit() bool {
	return p.hasDependency("svelte") && p.hasDependency("@sveltejs/kit")
}

// func (p *NodeProvider) isReact() bool {
// 	return p.hasDependency("react")
// }

// func (p *NodeProvider) isVue() bool {
// 	return p.hasDependency("vue")
// }

// func (p *NodeProvider) isPreact() bool {
// 	return p.hasDependency("preact")
// }

// func (p *NodeProvider) isLit() bool {
// 	return p.hasDependency("lit")
// }

// func (p *NodeProvider) isSolidJs() bool {
// 	return p.hasDependency("solid-js")
// }

// func (p *NodeProvider) isQwik() bool {
// 	return p.hasDependency("@builder.io/qwik")
// }
