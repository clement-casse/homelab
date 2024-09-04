package tailscale

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	// Namespace represents the K8S namespace in which the Tailscale Operator Helm release is deployed
	Namespace = "tailscale-system"

	// HelmChart is the name of the Helm Chart
	HelmChart = "tailscale-operator"

	// HelmRepoURL is the URL of the Tailscale Helm Repository
	HelmRepoURL = "https://pkgs.tailscale.com/helmcharts"

	// TailnetESCConfigKey is key of the pulumiconfig in the environment that stores the name of the Tailscale network
	TailnetESCConfigKey = "tailscaleTailnet"

	ClientIDESCConfigKey = "tailscaleK8SOperatorClientID"
	OAuthKeyESCSecretKey = "tailscaleK8SOperatorSecret"

	ChartCtxKey = "tsChart"
)

type DeployArgs struct {
	ClientID pulumi.String
	OAuthKey pulumi.StringOutput
}

type Deployment struct {
	Chart *helmv3.Release
}

// Deploy applies the Tailscale Helm chart to the given K8S provider
func Deploy(ctx *pulumi.Context, k8s *kubernetes.Provider, args *DeployArgs) (*Deployment, error) {
	rel, err := helmv3.NewRelease(ctx, "tailscale-operator", &helmv3.ReleaseArgs{
		Chart:           pulumi.String(HelmChart),
		Namespace:       pulumi.String(Namespace),
		CreateNamespace: pulumi.Bool(true),
		Atomic:          pulumi.Bool(true),
		RepositoryOpts: &helmv3.RepositoryOptsArgs{
			Repo: pulumi.String(HelmRepoURL),
		},
		Values: pulumi.ToMap(map[string]any{
			"installCRDs": true,
			"oauth": map[string]any{
				"clientId":     args.ClientID,
				"clientSecret": args.OAuthKey,
			},
			"operatorConfig": map[string]any{
				"defaultTags": []string{"tag:k8s-operator"},
			},
		}),
	}, pulumi.Provider(k8s))
	if err != nil {
		return nil, err
	}

	return &Deployment{
		Chart: rel,
	}, nil
}
