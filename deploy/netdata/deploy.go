package netdata

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"homelab/traefik"
)

var (
	// Namespace represents the K8S namespace in which the Netdata Helm release is deployed
	Namespace = "netdata"

	// RoomsESCConfigKey is key of the pulumiconfig in the environment that stores the IDs of the rooms for Netdata
	RoomsESCConfigKey = "netdataRooms"

	// TokenESCSecretKey is key of the pulumiconfig in the environment that stores the token used to register Netdata nodes
	TokenESCSecretKey = "netdataToken"
)

// DeployArgs is a struct that passes the arguments requiered to deploy the NetData monitoring service to the target K8S cluster
type DeployArgs struct {
	// Rooms
	Rooms pulumi.String

	// Token
	Token pulumi.StringOutput
}

// Deployment is the result of the Deploy funtion providing references to the pulumi resources, so that they can be
// referenced throughout the whole pulumi deployment.
type Deployment struct {
	// Release
	Release *helmv3.Release

	// TailscaleSvc
	TailscaleSvc *corev1.ServiceOutput
}

var (
	helmChart   = "netdata"
	helmRepoURL = "https://netdata.github.io/helmchart/"
)

// Deploy applies the netdata Helm Chart to the given K8S cluster.
func Deploy(ctx *pulumi.Context, k8s *kubernetes.Provider, args *DeployArgs) (*Deployment, error) {
	rel, err := helmv3.NewRelease(ctx, "netdata", &helmv3.ReleaseArgs{
		Chart:           pulumi.String(helmChart),
		Namespace:       pulumi.String(Namespace),
		CreateNamespace: pulumi.Bool(true),
		Atomic:          pulumi.Bool(true),
		RepositoryOpts: helmv3.RepositoryOptsArgs{
			Repo: pulumi.String(helmRepoURL),
		},
		Values: pulumi.ToMap(map[string]any{
			"image":     map[string]any{"tag": "edge"},
			"ingress":   map[string]any{"enabled": false},
			"restarter": map[string]any{"enabled": true},
			"parent": map[string]any{
				"env": map[string]any{"DO_NOT_TRACK": 1},
				"claiming": map[string]any{
					"enabled": true,
					"token":   args.Token,
					"rooms":   args.Rooms,
				},
				"database": map[string]any{
					"persistence":  true,
					"storageclass": "-", //TODO change this to SMB
				},
			},
			"child": map[string]any{
				"env": map[string]any{"DO_NOT_TRACK": 1},
				"claiming": map[string]any{
					"enabled": true,
					"token":   args.Token,
					"rooms":   args.Rooms,
				},
			},
			"k8sState": map[string]any{
				"env": map[string]any{"DO_NOT_TRACK": 1},
			},
		}),
	}, pulumi.Provider(k8s))
	if err != nil {
		return nil, err
	}

	// Retrieve the service exposing the UI of netdata.
	// Reference: https://www.pulumi.com/blog/full-access-to-helm-features-through-new-helm-release-resource-for-kubernetes/
	tSvc := pulumi.All(rel.Status.Namespace(), pulumi.String("netdata")).
		ApplyT(func(r []any) (*corev1.Service, error) {
			ns, name := r[0].(*string), r[1].(string)
			svc, err := corev1.GetService(ctx, "netdata-ui-svc", pulumi.ID(fmt.Sprintf("%s/%s", *ns, name)), nil)
			if err != nil {
				return nil, err
			}
			tsSvc, err := traefik.RegisterTailscaleSvc(ctx, "netdata", svc)
			if err != nil {
				return nil, err
			}
			return tsSvc.Service, nil
		}).(corev1.ServiceOutput)

	return &Deployment{
		Release:      rel,
		TailscaleSvc: &tSvc,
	}, nil
}
