package node

import (
	"regexp"
	"strings"

	"github.com/salamer/railpack/core/generate"
)

const (
	DefaultAstroOutputDirectory = "dist"
)

func (p *NodeProvider) isAstro(ctx *generate.GenerateContext) bool {
	hasAstroConfig := ctx.App.HasMatch("astro.config.mjs") || ctx.App.HasMatch("astro.config.ts")
	hasAstroBuildCommand := strings.Contains(strings.ToLower(p.packageJson.GetScript("build")), "astro build")

	return hasAstroConfig && hasAstroBuildCommand
}

func (p *NodeProvider) isAstroSPA(ctx *generate.GenerateContext) bool {
	if !p.isAstro(ctx) {
		return false
	}

	configFileContents := p.getAstroConfigFileContents(ctx)
	hasServerOutput := strings.Contains(configFileContents, "output: 'server'")
	hasAdapter := p.hasDependency("@astrojs/node") || p.hasDependency("@astrojs/vercel") || p.hasDependency("@astrojs/cloudflare") || p.hasDependency("@astrojs/netlify")

	return !hasServerOutput && !hasAdapter
}

func (p *NodeProvider) getAstroOutputDirectory(ctx *generate.GenerateContext) string {
	configFileContents := p.getAstroConfigFileContents(ctx)
	if configFileContents != "" {
		// Look for outDir in config
		outDirRegex := regexp.MustCompile(`outDir:\s*['"](.+?)['"]`)
		matches := outDirRegex.FindStringSubmatch(configFileContents)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return DefaultAstroOutputDirectory
}

func (p *NodeProvider) getAstroConfigFileContents(ctx *generate.GenerateContext) string {
	configFile := ""

	if ctx.App.HasMatch("astro.config.mjs") {
		contents, err := ctx.App.ReadFile("astro.config.mjs")
		if err == nil {
			configFile = contents
		}
	} else if ctx.App.HasMatch("astro.config.ts") {
		contents, err := ctx.App.ReadFile("astro.config.ts")
		if err == nil {
			configFile = contents
		}
	}

	return configFile
}

func (p *NodeProvider) getAstroEnvVars() map[string]string {
	envVars := map[string]string{
		"HOST": "0.0.0.0",
	}

	return envVars
}

func (p *NodeProvider) getAstroCache(ctx *generate.GenerateContext) string {
	return ctx.Caches.AddCache("astro", "node_modules/.astro")
}
