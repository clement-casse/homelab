package postgres

import (
	"fmt"
	"homelab/traefik"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:generate crd2pulumi --go --goName "crds" "https://raw.githubusercontent.com/zalando/postgres-operator/master/charts/postgres-operator/crds/postgresqls.yaml" "https://raw.githubusercontent.com/zalando/postgres-operator/master/charts/postgres-operator/crds/postgresteams.yaml" "https://raw.githubusercontent.com/zalando/postgres-operator/master/charts/postgres-operator/crds/operatorconfigurations.yaml"

var (
	// Namespace represents the K8S namespace in which the Postgres operator is deployed
	Namespace = "postgres-system"

	// ChartCtxKey is the value to be used to propagate the reference to the helm chart in the context
	ChartCtxKey = "pgOperator"
)

// DeployArgs is a struct that passes the arguments requiered to deploy the Postgres Operator and eventually
// the postgres-operator-ui too.
type DeployArgs struct {
	// InstallUI determines whether or not the postgres-operator-ui should be installed.
	InstallUI bool
}

// Deployment is the result of the Deploy funtion providing references to the pulumi resources, so that they can be
// referenced throughout the whole pulumi deployment.
type Deployment struct {
	// OperatorChart reference the Helm release of the postgres operator in the given Kubernetes cluster.
	OperatorChart *helmv3.Release

	// UIChart provides a reference to the Helm release of the postgres-operator-ui, it can be nil when the UI is not installed.
	UIChart *helmv3.Release

	// UITailscaleSvc
	UITailscaleSvc *corev1.ServiceOutput
}

var (
	helmChartOperator   = "postgres-operator"
	helmRepoURLOperator = "https://opensource.zalando.com/postgres-operator/charts/postgres-operator"
	helmChartUI         = "postgres-operator-ui"
	helmRepoURLUI       = "https://opensource.zalando.com/postgres-operator/charts/postgres-operator-ui"
)

// Deploy applies the Postgres-operator helm chart to the given K8S cluster.
func Deploy(ctx *pulumi.Context, k8s *kubernetes.Provider, args *DeployArgs) (*Deployment, error) {
	rel, err := helmv3.NewRelease(ctx, "postgres-operator", &helmv3.ReleaseArgs{
		Chart:           pulumi.String(helmChartOperator),
		Namespace:       pulumi.String(Namespace),
		CreateNamespace: pulumi.Bool(true),
		Atomic:          pulumi.Bool(true),
		RepositoryOpts: helmv3.RepositoryOptsArgs{
			Repo: pulumi.String(helmRepoURLOperator),
		},
		Values: pulumi.ToMap(map[string]any{}),
	}, pulumi.Provider(k8s))
	if err != nil {
		return nil, err
	}

	var uiRel *helmv3.Release
	var uiSvc *corev1.ServiceOutput

	if args.InstallUI {
		// Retrieve the service exposing the API of the Postgres operator which bears the name of the helm chart.
		// Reference: https://www.pulumi.com/blog/full-access-to-helm-features-through-new-helm-release-resource-for-kubernetes/
		opSvc := pulumi.All(rel.Status.Namespace(), rel.Status.Name()).
			ApplyT(func(r []any) (*corev1.Service, error) {
				ns, name := r[0].(*string), r[1].(*string)
				return corev1.GetService(ctx, "postgres-operator-svc", pulumi.ID(fmt.Sprintf("%s/%s", *ns, *name)), nil)
			}).(corev1.ServiceOutput)

		opSvcName := opSvc.Metadata().Name().Elem()
		opSvcPort := opSvc.Spec().Ports().Index(pulumi.Int(0)).Port()

		uiRel, err = helmv3.NewRelease(ctx, "postgres-operator-ui", &helmv3.ReleaseArgs{
			Chart:     pulumi.String(helmChartUI),
			Namespace: rel.Status.Namespace(),
			Atomic:    pulumi.Bool(true),
			RepositoryOpts: helmv3.RepositoryOptsArgs{
				Repo: pulumi.String(helmRepoURLUI),
			},
			Values: pulumi.ToMap(map[string]any{
				"envs": map[string]any{
					"appUrl":                   "",
					"operatorApiUrl":           pulumi.Sprintf("http://%s:%d", opSvcName, opSvcPort),
					"operatorClusterNameLabel": "",
					"targetNamespace":          "*",
				},
				"ingress": map[string]any{"enabled": false},
			}),
		}, pulumi.Provider(k8s), pulumi.DependsOn([]pulumi.Resource{rel}))
		if err != nil {
			return nil, err
		}

		// Retrieve the service exposing the UI of the postgres operator which bears the name of the helm chart.
		// Reference: https://www.pulumi.com/blog/full-access-to-helm-features-through-new-helm-release-resource-for-kubernetes/
		tSvc := pulumi.All(uiRel.Status.Namespace(), uiRel.Status.Name()).
			ApplyT(func(r []any) (*corev1.Service, error) {
				ns, name := r[0].(*string), r[1].(*string)
				svc, err := corev1.GetService(ctx, "postgres-operator-ui-svc", pulumi.ID(fmt.Sprintf("%s/%s", *ns, *name)), nil)
				if err != nil {
					return nil, err
				}
				tsSvc, err := traefik.RegisterTailscaleSvc(ctx, "postgres", svc)
				if err != nil {
					return nil, err
				}
				return tsSvc.Service, nil
			}).(corev1.ServiceOutput)

		uiSvc = &tSvc
	}

	return &Deployment{
		OperatorChart:  rel,
		UIChart:        uiRel,
		UITailscaleSvc: uiSvc,
	}, nil
}
