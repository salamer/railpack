package rust

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/salamer/railpack/core/generate"
	"github.com/salamer/railpack/core/plan"
	"github.com/salamer/railpack/internal/utils"
)

const (
	DEFAULT_RUST_VERSION = "1.85.1"
	CARGO_REGISTRY_CACHE = "/root/.cargo/registry"
	CARGO_GIT_CACHE      = "/root/.cargo/git"
	CARGO_TARGET_CACHE   = "target"
)

type RustProvider struct {
}

func (p *RustProvider) Name() string {
	return "rust"
}

func (p *RustProvider) Detect(ctx *generate.GenerateContext) (bool, error) {
	hasCargoToml := ctx.App.HasMatch("Cargo.toml")
	return hasCargoToml, nil
}

func (p *RustProvider) Initialize(ctx *generate.GenerateContext) error {
	return nil
}

func (p *RustProvider) Plan(ctx *generate.GenerateContext) error {
	miseStep := ctx.GetMiseStepBuilder()
	p.InstallMisePackages(ctx, miseStep)

	install := ctx.NewCommandStep("install")
	install.AddInputs([]plan.Input{
		plan.NewStepInput(miseStep.Name()),
	})
	p.Install(ctx, install)

	build := ctx.NewCommandStep("build")
	build.AddInputs([]plan.Input{
		plan.NewStepInput(miseStep.Name()),
		plan.NewStepInput(install.Name(), plan.InputOptions{
			Exclude: []string{"/app/"},
		}),
	})
	p.Build(ctx, build)

	maps.Copy(ctx.Deploy.Variables, p.GetRustEnvVars(ctx))
	ctx.Deploy.Inputs = []plan.Input{
		ctx.DefaultRuntimeInput(),
		plan.NewStepInput(build.Name(), plan.InputOptions{
			Include: []string{"/app/bin"},
		}),
	}
	ctx.Deploy.StartCmd = p.GetStartCommand(ctx)

	return nil
}

func (p *RustProvider) StartCommandHelp() string {
	return "To start your Rust application, Railpack will look for:\n\n" +
		"1. A Cargo.toml file in your project root\n\n" +
		"Your application will be compiled to a binary and started using `./bin/<binary>`"
}

func (p *RustProvider) GetStartCommand(ctx *generate.GenerateContext) string {
	target := p.getTarget(ctx)
	workspace := p.resolveCargoWorkspace(ctx)

	if target != "" {
		if workspace != "" {
			return fmt.Sprintf("./bin/%s", workspace)
		}

		return p.getStartBin(ctx)
	}

	if workspace != "" {
		return fmt.Sprintf("./bin/%s", workspace)
	}

	return p.getStartBin(ctx)
}

func (p *RustProvider) getStartBin(ctx *generate.GenerateContext) string {
	bins, err := p.getBins(ctx)
	if err != nil {
		return ""
	}

	if len(bins) == 0 {
		return ""
	}

	var bin string

	if len(bins) == 1 {
		bin = bins[0]
	} else if envBinName, _ := ctx.Env.GetConfigVariable("RUST_BIN"); envBinName != "" {
		for _, b := range bins {
			if b == envBinName {
				bin = b
				break
			}
		}
		if bin == "" {
			ctx.Logger.LogWarn("RUST_BIN environment variable set to '%s', but no matching binary found in Cargo.toml", envBinName)
		}
	} else {
		cargoToml, err := parseCargoTOML(ctx)
		if err == nil && cargoToml.Package.DefaultRun != "" {
			bin = cargoToml.Package.DefaultRun
		}
	}

	if bin == "" {
		return bin
	}

	binSuffix := p.getBinSuffix(ctx)
	return fmt.Sprintf("./bin/%s%s", bin, binSuffix)
}

func (p *RustProvider) getTarget(ctx *generate.GenerateContext) string {
	// Target may be defined in .config/cargo.toml
	if p.shouldMakeWasm32Wasi(ctx) {
		return "wasm32-wasi"
	}

	return ""
}

func (p *RustProvider) Install(ctx *generate.GenerateContext, install *generate.CommandStepBuilder) {
	install.AddCache(ctx.Caches.AddCache("cargo_registry", CARGO_REGISTRY_CACHE))
	install.AddCache(ctx.Caches.AddCache("cargo_git", CARGO_GIT_CACHE))
	install.AddCommands([]plan.Command{
		plan.NewCopyCommand("Cargo.toml*", "."),
		plan.NewCopyCommand("Cargo.lock*", "."),
	})

	buildCmd := "cargo build --release"
	dummyCmd := `echo "fn main() { }" > /app/src/main.rs && if grep -q "\[lib\]" Cargo.toml; then echo "fn main() { }" > /app/src/lib.rs; fi`
	target := p.getTarget(ctx)
	targetArg := ""
	targetPath := ""

	if target != "" {
		targetArg = fmt.Sprintf(" --target %s", target)
		targetPath = fmt.Sprintf("%s/", target)
		install.AddCommands([]plan.Command{
			plan.NewExecCommand(fmt.Sprintf("rustup target add %s", target)),
		})
	}

	// Don't do anything if we're in a workspace for now
	if workspace := p.resolveCargoWorkspace(ctx); workspace != "" {
		return
	}

	install.AddCommands([]plan.Command{
		plan.NewExecCommand(`mkdir -p src`),
		plan.NewExecShellCommand(dummyCmd, plan.ExecOptions{CustomName: "compile dependencies"}),
		plan.NewExecCommand(`cat /app/src/main.rs`),
		plan.NewExecCommand(fmt.Sprintf("%s%s", buildCmd, targetArg)),
		plan.NewExecCommand(fmt.Sprintf("rm -rf src target/%srelease/%s*", targetPath, p.getAppName(ctx))),
	})
}

func (p *RustProvider) Build(ctx *generate.GenerateContext, build *generate.CommandStepBuilder) {
	build.AddCommands([]plan.Command{
		plan.NewCopyCommand("."),
		plan.NewExecCommand("mkdir -p bin"),
	})

	buildCmd := "cargo build --release"
	binSuffix := p.getBinSuffix(ctx)
	target := p.getTarget(ctx)
	targetArg := ""
	targetPath := ""

	if target != "" {
		targetArg = fmt.Sprintf(" --target %s", target)
		targetPath = fmt.Sprintf("%s/", target)
		build.AddCommands([]plan.Command{
			plan.NewExecCommand(fmt.Sprintf("rustup target add %s", target)),
		})
	}

	workspace := p.resolveCargoWorkspace(ctx)

	if workspace != "" {
		buildCmd = fmt.Sprintf("%s --package %s%s", buildCmd, workspace, targetArg)
		build.AddCommands([]plan.Command{
			plan.NewExecCommand(buildCmd),
			plan.NewExecCommand(fmt.Sprintf("cp target/%srelease/%s%s bin", targetPath, workspace, binSuffix)),
		})
	} else {
		bins, err := p.getBins(ctx)
		if err != nil {
			return
		}

		if len(bins) > 0 {
			build.AddCommand(plan.NewExecCommand(fmt.Sprintf("%s%s", buildCmd, targetArg)))
			for _, bin := range bins {
				build.AddCommand(plan.NewExecCommand(fmt.Sprintf("cp target/%srelease/%s%s bin", targetPath, bin, binSuffix)))
			}
		}
	}

	appName := p.getAppName(ctx)
	if appName != "" {
		build.AddCache(ctx.Caches.AddCache("cargo_target", CARGO_TARGET_CACHE))
	}
}

func (p *RustProvider) getBinSuffix(ctx *generate.GenerateContext) string {
	if p.shouldMakeWasm32Wasi(ctx) {
		return ".wasm"
	}
	return ""
}

func (p *RustProvider) getBins(ctx *generate.GenerateContext) ([]string, error) {
	var bins []string

	name := p.getAppName(ctx)
	if name != "" {
		if ctx.App.HasMatch("src/main.rs") {
			bins = append(bins, name)
		}
	}

	if ctx.App.HasMatch("src/bin") {
		findBins, err := ctx.App.FindFiles("src/bin/*")
		if err != nil {
			return nil, err
		}

		for _, binPath := range findBins {
			if binPath == "" {
				continue
			}

			binPathParts := strings.Split(binPath, "/")
			if len(binPathParts) == 0 {
				continue
			}

			filename := binPathParts[len(binPathParts)-1]
			parts := strings.Split(filename, ".")
			if len(parts) <= 1 {
				continue
			}

			binName := strings.Join(parts[:len(parts)-1], ".")
			bins = append(bins, binName)
		}
	}

	if len(bins) == 0 {
		return nil, nil
	}

	return bins, nil
}

func (p *RustProvider) getAppName(ctx *generate.GenerateContext) string {
	tomlFile, err := parseCargoTOML(ctx)
	if err != nil {
		return ""
	}

	if tomlFile.Package.Name != "" {
		return tomlFile.Package.Name
	}

	return ""
}

func (p *RustProvider) GetRustEnvVars(ctx *generate.GenerateContext) map[string]string {
	return map[string]string{
		"ROCKET_ADDRESS": "0.0.0.0", // Allows Rocket apps to accept non-local connections
	}
}

func (p *RustProvider) InstallMisePackages(ctx *generate.GenerateContext, miseStep *generate.MiseStepBuilder) {
	rust := miseStep.Default("rust", DEFAULT_RUST_VERSION)

	cargoToml, _ := parseCargoTOML(ctx)
	if cargoToml != nil {
		// Fall back to the edition
		switch cargoToml.Package.Edition {
		case "2015":
			// https://doc.rust-lang.org/edition-guide/rust-2015/index.html
			// >= 1.0.0
			miseStep.Version(rust, "1.30.0", "Cargo.toml")
		case "2018":
			// https://doc.rust-lang.org/edition-guide/rust-2021/index.html
			// >= 1.31.0
			miseStep.Version(rust, "1.55.0", "Cargo.toml")
		case "2021":
			// https://doc.rust-lang.org/edition-guide/rust-2021/index.html
			// >= 1.56.0
			miseStep.Version(rust, "1.84.0", "Cargo.toml")
		case "2024":
			// https://doc.rust-lang.org/edition-guide/rust-2024/index.html
			// >= 1.85.0
			miseStep.Version(rust, "1.85.1", "Cargo.toml")
		}
	}

	if envVersion, varName := ctx.Env.GetConfigVariable("RUST_VERSION"); envVersion != "" {
		miseStep.Version(rust, envVersion, varName)
	}

	for _, filename := range []string{"rust-version.txt", ".rust-version"} {
		if content, err := ctx.App.ReadFile(filename); err == nil {
			if version := strings.TrimSpace(utils.ExtractSemverVersion(content)); version != "" {
				miseStep.Version(rust, version, filename)
			}
		}
	}

	if cargoToml != nil {
		if cargoToml.Package.RustVersion != "" {
			// Newer versions of Rust allow the `rust-version` field in Cargo.toml
			if version := utils.ExtractSemverVersion(cargoToml.Package.RustVersion); version != "" {
				miseStep.Version(rust, version, "Cargo.toml")
			}
		}
	}

	if toolchain, err := parseRustToolchain(ctx); err == nil {
		if version := utils.ExtractSemverVersion(toolchain.Toolchain.Version); version != "" {
			miseStep.Version(rust, version, "rust-toolchain.toml")
		}
	}
}

var wasmRegex = regexp.MustCompile(`target\s*=\s*"wasm32-wasi"`)

func (p *RustProvider) shouldMakeWasm32Wasi(ctx *generate.GenerateContext) bool {
	matches := ctx.App.FindFilesWithContent(".cargo/config.toml", wasmRegex)
	return len(matches) > 0
}

func (p *RustProvider) resolveCargoWorkspace(ctx *generate.GenerateContext) string {
	// First check for environment variable override
	if name, _ := ctx.Env.GetConfigVariable("CARGO_WORKSPACE"); name != "" {
		return name
	}

	// Then check for workspace in Cargo.toml
	cargoToml, err := parseCargoTOML(ctx)
	if err == nil && cargoToml != nil && cargoToml.Workspace.Members != nil {
		if binary, err := p.findBinaryInWorkspace(ctx, cargoToml.Workspace); err == nil && binary != "" {
			return binary
		}
	}

	return ""
}

func (p *RustProvider) findBinaryInWorkspace(ctx *generate.GenerateContext, workspace WorkspaceConfig) (string, error) {
	findBinary := func(member string) (string, error) {
		path := fmt.Sprintf("%s/Cargo.toml", member)
		var manifest CargoTOML
		if err := ctx.App.ReadTOML(path, &manifest); err != nil {
			return "", err
		}

		if manifest.Package.Name == "" {
			return "", nil
		}

		// Check for src/main.rs which definitely indicates a binary
		hasMainRs := ctx.App.HasMatch(fmt.Sprintf("%s/src/main.rs", member))
		if hasMainRs {
			return manifest.Package.Name, nil
		}

		// Check for binaries in src/bin/
		hasBinDir := ctx.App.HasMatch(fmt.Sprintf("%s/src/bin", member))
		if hasBinDir {
			return manifest.Package.Name, nil
		}

		// Check for bin entries in the manifest
		if len(manifest.Bin) > 0 {
			return manifest.Package.Name, nil
		}

		// If no lib.rs exists, it might be a binary
		hasLibRs := ctx.App.HasMatch(fmt.Sprintf("%s/src/lib.rs", member))
		if !hasLibRs {
			return manifest.Package.Name, nil
		}

		return "", nil
	}

	// First check default members that aren't excluded
	for _, defaultMember := range workspace.DefaultMembers {
		if slices.Contains(workspace.ExcludeMembers, defaultMember) {
			continue
		}

		if strings.Contains(defaultMember, "*") || strings.Contains(defaultMember, "?") {
			dirs, err := ctx.App.FindDirectories(defaultMember)
			if err != nil {
				return "", err
			}

			for _, dir := range dirs {
				binary, err := findBinary(dir)
				if err == nil && binary != "" {
					return binary, nil
				}
			}
		} else {
			binary, err := findBinary(defaultMember)
			if err == nil && binary != "" {
				return binary, nil
			}
		}
	}

	// Then check all members that aren't excluded
	for _, member := range workspace.Members {
		if slices.Contains(workspace.ExcludeMembers, member) {
			continue
		}

		if strings.Contains(member, "*") || strings.Contains(member, "?") {
			dirs, err := ctx.App.FindDirectories(member)
			if err != nil {
				return "", err
			}

			for _, dir := range dirs {
				binary, err := findBinary(dir)
				if err == nil && binary != "" {
					return binary, nil
				}
			}
		} else {
			binary, err := findBinary(member)
			if err == nil && binary != "" {
				return binary, nil
			}
		}
	}

	return "", nil
}

// parseCargoTOML parses a Cargo.toml file
func parseCargoTOML(ctx *generate.GenerateContext) (*CargoTOML, error) {
	var cargoToml *CargoTOML
	if err := ctx.App.ReadTOML("Cargo.toml", &cargoToml); err != nil {
		return nil, err
	}

	return cargoToml, nil
}

// See https://doc.rust-lang.org/cargo/reference/manifest.html
type CargoTOML struct {
	Package           PackageInfo     `toml:"package"`
	Dependencies      map[string]any  `toml:"dependencies,omitempty"`
	DevDependencies   map[string]any  `toml:"dev-dependencies,omitempty"`
	BuildDependencies map[string]any  `toml:"build-dependencies,omitempty"`
	Lib               LibConfig       `toml:"lib,omitempty"`
	Bin               []BinConfig     `toml:"bin,omitempty"`
	Workspace         WorkspaceConfig `toml:"workspace,omitempty"`
}

type PackageInfo struct {
	Name          string   `toml:"name"`
	Version       string   `toml:"version"`
	RustVersion   string   `toml:"rust-version,omitempty"`
	Authors       []string `toml:"authors,omitempty"`
	Edition       string   `toml:"edition,omitempty"`
	Description   string   `toml:"description,omitempty"`
	License       string   `toml:"license,omitempty"`
	Repository    string   `toml:"repository,omitempty"`
	Homepage      string   `toml:"homepage,omitempty"`
	Documentation string   `toml:"documentation,omitempty"`
	Keywords      []string `toml:"keywords,omitempty"`
	Categories    []string `toml:"categories,omitempty"`
	Readme        string   `toml:"readme,omitempty"`
	Exclude       []string `toml:"exclude,omitempty"`
	Include       []string `toml:"include,omitempty"`
	Publish       bool     `toml:"publish,omitempty"`
	DefaultRun    string   `toml:"default-run,omitempty"`
}

type Dependency struct {
	Version         string   `toml:"version,omitempty"`
	Path            string   `toml:"path,omitempty"`
	Git             string   `toml:"git,omitempty"`
	Branch          string   `toml:"branch,omitempty"`
	Tag             string   `toml:"tag,omitempty"`
	Rev             string   `toml:"rev,omitempty"`
	Features        []string `toml:"features,omitempty"`
	Optional        bool     `toml:"optional,omitempty"`
	DefaultFeatures bool     `toml:"default-features,omitempty"`
	Package         string   `toml:"package,omitempty"`
}

type LibConfig struct {
	Name      string   `toml:"name,omitempty"`
	Path      string   `toml:"path,omitempty"`
	CrateType []string `toml:"crate-type,omitempty"`
	ProcMacro bool     `toml:"proc-macro,omitempty"`
	Harness   bool     `toml:"harness,omitempty"`
	Test      bool     `toml:"test,omitempty"`
	DocTest   bool     `toml:"doctest,omitempty"`
	Bench     bool     `toml:"bench,omitempty"`
	Doc       bool     `toml:"doc,omitempty"`
	Plugin    bool     `toml:"plugin,omitempty"`
}

type BinConfig struct {
	Name     string `toml:"name,omitempty"`
	Path     string `toml:"path,omitempty"`
	Test     bool   `toml:"test,omitempty"`
	Bench    bool   `toml:"bench,omitempty"`
	Doc      bool   `toml:"doc,omitempty"`
	Plugin   bool   `toml:"plugin,omitempty"`
	Harness  bool   `toml:"harness,omitempty"`
	Required bool   `toml:"required,omitempty"`
}

type Profile struct {
	Opt              string            `toml:"opt-level,omitempty"`
	Debug            bool              `toml:"debug,omitempty"`
	Rpath            bool              `toml:"rpath,omitempty"`
	LtoFlags         []string          `toml:"lto,omitempty"`
	Debug_assertions bool              `toml:"debug-assertions,omitempty"`
	Codegen          map[string]string `toml:"codegen-units,omitempty"`
	Panic            string            `toml:"panic,omitempty"`
	Incremental      bool              `toml:"incremental,omitempty"`
	Overflow_checks  bool              `toml:"overflow-checks,omitempty"`
}

type WorkspaceConfig struct {
	Members        []string `toml:"members,omitempty"`
	ExcludeMembers []string `toml:"exclude,omitempty"`
	DefaultMembers []string `toml:"default-members,omitempty"`
	Resolver       string   `toml:"resolver,omitempty"`
}

type RustToolchain struct {
	// The toolchain specification
	Toolchain ToolchainSpec `toml:"toolchain"`
}

type ToolchainSpec struct {
	Channel    string   `toml:"channel"`
	Version    string   `toml:"version,omitempty"`
	Components []string `toml:"components,omitempty"`
	Targets    []string `toml:"targets,omitempty"`
	Profile    string   `toml:"profile,omitempty"`
}

// parseRustToolchain parses a rust-toolchain.toml file
func parseRustToolchain(ctx *generate.GenerateContext) (*RustToolchain, error) {
	var toolchain RustToolchain

	// Try to read rust-toolchain.toml first (newer format)
	err := ctx.App.ReadTOML("rust-toolchain.toml", &toolchain)
	if err == nil {
		return &toolchain, nil
	}

	// Fall back to rust-toolchain file (older format)
	content, err := ctx.App.ReadFile("rust-toolchain")
	if err != nil {
		return nil, fmt.Errorf("no rust-toolchain files found: %w", err)
	}

	// Old format just contains the channel/version as plain text
	channel := strings.TrimSpace(string(content))
	return &RustToolchain{
		Toolchain: ToolchainSpec{
			Channel: channel,
		},
	}, nil
}
