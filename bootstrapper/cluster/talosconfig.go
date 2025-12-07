package cluster

import (
	"encoding/base64"
	"os"
	"text/template"
)

const talosconfigTemplate = `context: {{.Context}}
contexts:
    {{.Context}}:
        endpoints:
{{- range .Endpoints}}
            - {{.}}
{{- end}}
        nodes:
{{- range .Nodes}}
            - {{.}}
{{- end}}
        ca: "{{.CA}}"
        crt: "{{.Crt}}"
        key: "{{.Key}}"
`

type TalosconfigData struct {
	Context   string
	Endpoints []string
	Nodes     []string
	CA        string
	Crt       string
	Key       string
}

func (c Config) generateTalosconfig(outputPath string) error {
	tmpl, err := template.New("talosconfig").Parse(talosconfigTemplate)
	if err != nil {
		return err
	}

	data := TalosconfigData{
		Context:   c.clusterName,
		Endpoints: c.controlPlaneAddresses(),
		Nodes:     c.getAllNodeAddresses(),
		CA:        base64.StdEncoding.EncodeToString([]byte(c.secrets.OSCert)),
		Crt:       base64.StdEncoding.EncodeToString([]byte(c.secrets.OSAdminCert)),
		Key:       base64.StdEncoding.EncodeToString([]byte(c.secrets.OSAdminKey)),
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

func (c Config) getAllNodeAddresses() []string {
	return append(c.controlPlaneAddresses(), c.workerAddresses()...)
}

func (c Config) controlPlaneAddresses() []string {
	addresses := make([]string, 0, len(c.controlPlanes))
	for _, cp := range c.controlPlanes {
		addresses = append(addresses, cp.Address)
	}
	return addresses
}

func (c Config) workerAddresses() []string {
	addresses := make([]string, 0, len(c.workers))
	for _, worker := range c.workers {
		addresses = append(addresses, worker.Address)
	}
	return addresses
}
