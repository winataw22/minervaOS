package zosapi

import (
	"context"
	"os/exec"
	"strings"
)

func (g *ZosAPI) systemVersionHandler(ctx context.Context, payload []byte) (interface{}, error) {
	output, err := exec.CommandContext(ctx, "zinit", "-V").CombinedOutput()
	var zInitVer string
	if err != nil {
		zInitVer = err.Error()
	} else {
		zInitVer = strings.TrimSpace(strings.TrimPrefix(string(output), "zinit"))
	}

	version := struct {
		ZOS   string `json:"zos"`
		ZInit string `json:"zinit"`
	}{
		ZOS:   g.versionMonitorStub.GetVersion(ctx).String(),
		ZInit: zInitVer,
	}

	return version, nil
}

func (g *ZosAPI) systemDMIHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.oracle.DMI()
}

func (g *ZosAPI) systemHypervisorHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.oracle.GetHypervisor()
}

func (g *ZosAPI) systemDiagnosticsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.diagnosticsManager.GetSystemDiagnostics(ctx)
}
