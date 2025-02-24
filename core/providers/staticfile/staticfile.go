package staticfile

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/railwayapp/railpack/core/generate"
	"github.com/railwayapp/railpack/core/plan"
)

//go:embed Caddyfile.template
var caddyfileTemplate string

const (
	StaticfileConfigName = "Staticfile"
	CaddyfilePath        = "Caddyfile"
)

type StaticfileConfig struct {
	RootDir string `yaml:"root"`
}

type StaticfileProvider struct {
	RootDir string
}

func (p *StaticfileProvider) Name() string {
	return "staticfile"
}

func (p *StaticfileProvider) Initialize(ctx *generate.GenerateContext) error {
	rootDir, err := getRootDir(ctx)
	if err != nil {
		return err
	}

	p.RootDir = rootDir

	return nil
}

func (p *StaticfileProvider) Detect(ctx *generate.GenerateContext) (bool, error) {
	rootDir, err := getRootDir(ctx)
	if rootDir != "" && err == nil {
		return true, nil
	}

	return false, nil
}

func (p *StaticfileProvider) Plan(ctx *generate.GenerateContext) error {
	miseStep := ctx.GetMiseStepBuilder()
	miseStep.Default("caddy", "latest")

	setup := ctx.NewCommandStep("setup")
	setup.AddInput(plan.NewStepInput(miseStep.Name()))
	err := p.Setup(ctx, setup)
	if err != nil {
		return err
	}

	ctx.Deploy.Inputs = append(ctx.Deploy.Inputs, []plan.Input{
		ctx.DefaultRuntimeInput(),
		plan.NewStepInput(miseStep.Name(), plan.InputOptions{
			Include: miseStep.GetOutputPaths(),
		}),
		plan.NewStepInput(setup.Name(), plan.InputOptions{
			Include: []string{"."},
		}),
		plan.NewLocalInput("."),
	}...)

	ctx.Deploy.StartCmd = p.CaddyStartCommand(ctx)

	return nil
}

func (p *StaticfileProvider) CaddyStartCommand(ctx *generate.GenerateContext) string {
	return "caddy run --config " + CaddyfilePath + " --adapter caddyfile 2>&1"
}

func (p *StaticfileProvider) Setup(ctx *generate.GenerateContext, setup *generate.CommandStepBuilder) error {
	data := map[string]interface{}{
		"STATIC_FILE_ROOT": p.RootDir,
	}

	caddyfile, err := p.getCaddyfile(data)
	if err != nil {
		return err
	}

	setup.AddCommands([]plan.Command{
		plan.NewFileCommand(CaddyfilePath, "Caddyfile"),
		plan.NewExecCommand("caddy fmt --overwrite Caddyfile"),
	})

	setup.Assets = map[string]string{
		"Caddyfile": caddyfile,
	}

	return nil
}

func (p *StaticfileProvider) getCaddyfile(data map[string]interface{}) (string, error) {
	tmpl, err := template.New("Caddyfile").Parse(caddyfileTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func getRootDir(ctx *generate.GenerateContext) (string, error) {
	if rootDir, _ := ctx.Env.GetConfigVariable("STATIC_FILE_ROOT"); rootDir != "" {
		return rootDir, nil
	}

	staticfileConfig, err := getStaticfileConfig(ctx)
	if staticfileConfig != nil && err == nil {
		return staticfileConfig.RootDir, nil
	}

	if ctx.App.HasMatch("public") {
		return "public", nil
	} else if ctx.App.HasMatch("index.html") {
		return ".", nil
	}

	return "", fmt.Errorf("no static file root dir found")
}

func getStaticfileConfig(ctx *generate.GenerateContext) (*StaticfileConfig, error) {
	if !ctx.App.HasMatch(StaticfileConfigName) {
		return nil, nil
	}

	staticfileData := StaticfileConfig{}
	if err := ctx.App.ReadYAML(StaticfileConfigName, &staticfileData); err != nil {
		return nil, err
	}

	return &staticfileData, nil
}
