package generate

import (
	"fmt"
	"sort"
	"strings"

	a "github.com/railwayapp/railpack/core/app"
	"github.com/railwayapp/railpack/core/mise"
	"github.com/railwayapp/railpack/core/plan"
	"github.com/railwayapp/railpack/core/resolver"
)

const (
	MisePackageStepName = "packages:mise"
)

type MiseStepBuilder struct {
	DisplayName           string
	Resolver              *resolver.Resolver
	SupportingAptPackages []string
	MisePackages          []*resolver.PackageRef
	SupportingMiseFiles   []string
	Assets                map[string]string
	DependsOn             []string
	Outputs               *[]string

	app *a.App
	env *a.Environment
}

func (c *GenerateContext) newMiseStepBuilder() *MiseStepBuilder {
	step := &MiseStepBuilder{
		DisplayName:           MisePackageStepName,
		Resolver:              c.Resolver,
		MisePackages:          []*resolver.PackageRef{},
		SupportingAptPackages: []string{},
		Assets:                map[string]string{},
		DependsOn:             []string{},
		Outputs:               &[]string{"/mise/shims", "/mise/installs"},
		app:                   c.App,
		env:                   c.Env,
	}

	c.Steps = append(c.Steps, step)

	return step
}

func (b *MiseStepBuilder) AddSupportingAptPackage(name string) {
	b.SupportingAptPackages = append(b.SupportingAptPackages, name)
}

func (b *MiseStepBuilder) Default(name string, defaultVersion string) resolver.PackageRef {
	for _, pkg := range b.MisePackages {
		if pkg.Name == name {
			return *pkg
		}
	}

	pkg := b.Resolver.Default(name, defaultVersion)
	b.MisePackages = append(b.MisePackages, &pkg)
	return pkg
}

func (b *MiseStepBuilder) Version(name resolver.PackageRef, version string, source string) {
	b.Resolver.Version(name, version, source)
}

func (b *MiseStepBuilder) Name() string {
	return b.DisplayName
}

func (b *MiseStepBuilder) Build(options *BuildStepOptions) (*plan.Step, error) {
	step := plan.NewStep(b.DisplayName)

	if len(b.MisePackages) == 0 {
		return step, nil
	}

	step.DependsOn = b.DependsOn

	miseCache := options.Caches.AddCache("mise", "/mise/cache")

	// Install mise
	step.AddCommands([]plan.Command{
		plan.NewVariableCommand("MISE_DATA_DIR", "/mise"),
		plan.NewVariableCommand("MISE_CONFIG_DIR", "/mise"),
		plan.NewVariableCommand("MISE_INSTALL_PATH", "/usr/local/bin/mise"),
		plan.NewVariableCommand("MISE_CACHE_DIR", "/mise/cache"),
		options.NewAptInstallCommand([]string{"curl", "ca-certificates", "git"}),
		plan.NewExecCommand("sh -c 'curl -fsSL https://mise.run | sh'",
			plan.ExecOptions{
				CustomName: "install mise",
				Caches:     []string{miseCache},
			}),
	})

	// Add user mise config files if they exist
	supportingMiseConfigFiles := b.GetSupportingMiseConfigFiles(b.app.Source)
	for _, file := range supportingMiseConfigFiles {
		step.AddCommands([]plan.Command{
			plan.NewCopyCommand(file, "/app/"+file),
		})
	}

	// Setup apt commands
	if len(b.SupportingAptPackages) > 0 {
		step.AddCommands([]plan.Command{
			options.NewAptInstallCommand(b.SupportingAptPackages),
		})
	}

	// Setup mise commands
	if len(b.MisePackages) > 0 {
		packagesToInstall := make(map[string]string)
		for _, pkg := range b.MisePackages {
			resolved, ok := options.ResolvedPackages[pkg.Name]
			if ok && resolved.ResolvedVersion != nil {
				packagesToInstall[pkg.Name] = *resolved.ResolvedVersion
			}
		}

		miseToml, err := mise.GenerateMiseToml(packagesToInstall)
		if err != nil {
			return nil, fmt.Errorf("failed to generate mise.toml: %w", err)
		}

		b.Assets["mise.toml"] = miseToml

		pkgNames := make([]string, 0, len(packagesToInstall))
		for k := range packagesToInstall {
			pkgNames = append(pkgNames, k)
		}
		sort.Strings(pkgNames)

		step.AddCommands([]plan.Command{
			plan.NewFileCommand("/etc/mise/config.toml", "mise.toml", plan.FileOptions{
				CustomName: "create mise config",
			}),
			plan.NewExecCommand("sh -c 'mise trust -a && mise install'", plan.ExecOptions{
				CustomName: "install mise packages: " + strings.Join(pkgNames, ", "),
				Caches:     []string{miseCache},
			}),
		})
	}

	// Packages installed have binaries available at /mise/installs/{package}/{version}/bin
	// We need to add these to the PATH
	for _, pkg := range b.MisePackages {
		resolved, ok := options.ResolvedPackages[pkg.Name]
		if !ok || resolved.ResolvedVersion == nil {
			continue
		}

		version := *resolved.ResolvedVersion

		step.AddCommands([]plan.Command{
			plan.NewPathCommand("/mise/installs/" + pkg.Name + "/" + version + "/bin"),
		})
	}

	step.Assets = b.Assets
	step.Outputs = b.Outputs
	step.UseSecrets = &[]bool{false}[0]

	return step, nil
}

var miseConfigFiles = []string{
	"mise.toml",
	".python-version",
	".nvmrc",
}

func (b *MiseStepBuilder) GetSupportingMiseConfigFiles(path string) []string {
	files := []string{}

	for _, file := range miseConfigFiles {
		if b.app.HasMatch(file) {
			files = append(files, file)
		}
	}

	return files
}
