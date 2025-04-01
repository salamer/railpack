package procfile

import "github.com/salamer/railpack/core/generate"

type ProcfileProvider struct{}

func (p *ProcfileProvider) Name() string {
	return "procfile"
}

func (p *ProcfileProvider) Plan(ctx *generate.GenerateContext) (bool, error) {
	if _, err := ctx.App.ReadFile("Procfile"); err != nil {
		return false, nil
	}

	parsedProcfile := map[string]string{}
	if err := ctx.App.ReadYAML("Procfile", &parsedProcfile); err != nil {
		return false, err
	}

	webCommand := parsedProcfile["web"]
	workerCommand := parsedProcfile["worker"]

	if webCommand != "" {
		ctx.Logger.LogInfo("Found web command in Procfile")
		ctx.Deploy.StartCmd = webCommand
	} else if workerCommand != "" {
		ctx.Logger.LogInfo("Found worker command in Procfile")
		ctx.Deploy.StartCmd = workerCommand
	}

	return false, nil
}
