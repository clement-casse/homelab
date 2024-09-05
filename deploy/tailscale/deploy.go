package tailscale

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	// Namespace represents the K8S namespace in which the Tailscale Operator Helm release is deployed
	Namespace = "tailscale-system"

	// TailnetESCConfigKey is key of the pulumiconfig in the environment that stores the name of the Tailscale network
	TailnetESCConfigKey = "tailscaleTailnet"

	// ClientIDESCConfigKey is key of the pulumiconfig in the environment that stores the Tailscale ClientID for the K8S operator
	ClientIDESCConfigKey = "tailscaleK8SOperatorClientID"

	// OAuthKeyESCSecretKey is key of the pulumiconfig in the environment that stores the Tailscale Secret for the K8S operator
	OAuthKeyESCSecretKey = "tailscaleK8SOperatorSecret"

	// ChartCtxKey represents the context key to store the *helmv3.Release for the Tailscale Operator
	ChartCtxKey = "tsChart"
)

// DeployArgs is a struct that passes the arguments requiered to deploy the Tailscale Operator to the target K8S cluster
type DeployArgs struct {
	// ClientID
	ClientID pulumi.String

	// OAuthKey
	OAuthKey pulumi.StringOutput
}

// Deployment is the result of the Deploy funtion providing references to the pulumi resources, so that they can be
// referenced throughout the whole pulumi deployment.
type Deployment struct {
	// Chart
	Chart *helmv3.Release
}

var (
	helmChart   = "tailscale-operator"
	helmRepoURL = "https://pkgs.tailscale.com/helmcharts"
)

// Deploy applies the Tailscale Helm chart to the given K8S provider
func Deploy(ctx *pulumi.Context, k8s *kubernetes.Provider, args *DeployArgs) (*Deployment, error) {
	rel, err := helmv3.NewRelease(ctx, "tailscale-operator", &helmv3.ReleaseArgs{
		Chart:           pulumi.String(helmChart),
		Namespace:       pulumi.String(Namespace),
		CreateNamespace: pulumi.Bool(true),
		Atomic:          pulumi.Bool(true),
		RepositoryOpts: &helmv3.RepositoryOptsArgs{
			Repo: pulumi.String(helmRepoURL),
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
