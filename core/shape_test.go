package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIaCFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Existing Terraform patterns
		{"main.tf", true},
		{"variables.tfvars", true},
		{"sub/resource.hcl", true},

		// Docker patterns
		{"Dockerfile", true},
		{"api.dockerfile", true},
		{"docker-compose.yml", true},
		{"docker-compose.yaml", true},
		{"Containerfile", true},

		// Kubernetes patterns
		{"manifests/pod.yaml", true},
		{"deploy/deployment.json", true},
		{"kube/service.yml", true},
		{"kubernetes/ingress.yaml", true},
		{"k8s/pvc.json", true},

		// Helm patterns
		{"Chart.yaml", true},
		{"values.yaml", true},
		{"charts/my-app/templates/service.yaml", true},

		// Ansible patterns
		{"site.yml", true},
		{"playbook.yml", true},
		{"ansible.cfg", true},
		{"group_vars/all.yml", true},
		{"host_vars/db.yaml", true},
		{"inventory/hosts", false}, // Currently we only check YAML/JSON for these paths
		{"roles/web/tasks/main.yml", true},

		// Generic Infrastructure patterns
		{"infra/config.yaml", true},
		{"infrastructure/setup.json", true},
		{"ops/build.yml", true},
		{"env/prod/vars.yaml", true},

		// Generic Suffixes
		{"app-deployment.yaml", true},
		{"database-stack.json", true},
		{"resource-provision.yml", true},
		{"web.containerfile", true},

		// Other Cloud patterns
		{"Pulumi.yaml", true},
		{"Vagrantfile", true},

		// Non-IaC files
		{"main.go", false},
		{"src/utils.js", false},
		{"README.md", false},
		{"config.json", false}, // JSON not in a known path
		{"site.html", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, isIaCFile(tt.path), "isIaCFile(%q) mismatch", tt.path)
		})
	}
}
