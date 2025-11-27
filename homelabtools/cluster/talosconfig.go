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
		Endpoints: []string{c.controlPlane.Address},
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
	addresses := []string{c.controlPlane.Address}
	for _, worker := range c.workers {
		addresses = append(addresses, worker.Address)
	}
	return addresses
}
