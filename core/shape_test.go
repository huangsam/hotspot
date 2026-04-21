package core

import (
	"testing"

	"github.com/huangsam/hotspot/schema"
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

		// Chef patterns
		{"Cheffile", true},
		{"Berksfile", true},
		{"chef/recipe.rb", true},
		{"recipes/default.rb", true},

		// Puppet patterns
		{"Puppetfile", true},
		{"hiera.yaml", true},
		{"manifests/site.pp", true},
		{"puppet/module.pp", true},

		// CloudFormation patterns
		{"cloudformation/template.yaml", true},
		{"cfn/stack.json", true},
		{"my-stack.cf.yaml", true},
		{"infra.cf.yml", true},
		{"template.cf.json", true},

		// Serverless Framework
		{"serverless.yml", true},
		{"serverless.yaml", true},
		{"serverless/function.yml", true},

		// Azure Bicep & ARM
		{"main.bicep", true},
		{"azuredeploy.json", true},
		{"bicep/network.bicep", true},

		// Terragrunt & SAM & Packer
		{"terragrunt.hcl", true},
		{"samconfig.toml", true},
		{"sam/parameters.toml", true},
		{"packer.json", true},
		{"packer/ubuntu.pkr.hcl", true},

		// GitOps & Others
		{"gitops/app.yaml", true},
		{"flux/sync.yaml", true},
		{"argo/app.json", true},
		{"ansible/playbook.jinja", true},

		// Helm patterns (extended)
		{"helmfile.yaml", true},
		{"helm/values.yaml", true},

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

func TestComputeRepoShape(t *testing.T) {
	files := []string{
		"main.go",
		"utils.go",
		"infra/main.tf",
		"deployment.yaml", // Not IaC because not in known path and no special suffix
		"app.cf.yaml",     // IaC
	}

	output := &schema.AggregateOutput{
		FileStats: map[string]*schema.FileAggregation{
			"main.go": {
				Commits: 10,
				Churn:   100,
				Contributors: map[string]schema.Metric{
					"user1": 10,
				},
			},
			"utils.go": {
				Commits: 5,
				Churn:   50,
				Contributors: map[string]schema.Metric{
					"user1": 5,
					"user2": 2,
				},
			},
		},
	}

	shape := ComputeRepoShape(files, output)

	assert.Equal(t, 5, shape.FileCount)
	assert.Equal(t, 15.0, shape.TotalCommits)
	assert.Equal(t, 2, shape.UniqueContributors)
	assert.Equal(t, 75.0, shape.AvgChurnPerFile)      // (100+50)/2
	assert.InDelta(t, 0.4, shape.IaCFileRatio, 0.001) // infra/main.tf and app.cf.yaml are IaC. 2/5 = 0.4
	assert.Equal(t, schema.PresetInfra, shape.RecommendedPreset)
}

func BenchmarkIsIaCFile(b *testing.B) {
	cases := []string{
		"main.tf",                              // Strong match
		"internal/infra/config.yaml",           // Contextual match
		"app-deployment.yaml",                  // Heuristic match
		"src/pkg/core/logic/processor/main.go", // Non-match (The "Hot" path)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, c := range cases {
			_ = isIaCFile(c)
		}
	}
}
