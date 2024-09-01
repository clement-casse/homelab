package netdata

import (
	"homelab/traefik"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	// Namespace represents the K8S namespace in which the Netdata Helm release is deployed
	Namespace = "netdata"

	// HelmChart is the name of the Helm Chart
	HelmChart = "netdata"

	// HelmRepoURL is the URL of the Netdata Helm Repository
	HelmRepoURL = "https://netdata.github.io/helmchart/"

	RoomsESCConfigKey = "netdataRooms"
	TokenESCSecretKey = "netdataToken"
)

type DeployArgs struct {
	Rooms          pulumi.String
	Token          pulumi.StringOutput
	TailscaleChart *helmv3.Release
}

type Deployment struct {
	Namespace *corev1.Namespace
	Chart     *helmv3.Chart
}

// Deploy applies the netdata Helm Chart to the given K8S cluster.
func Deploy(ctx *pulumi.Context, k8s *kubernetes.Provider, args *DeployArgs) (*Deployment, error) {
	ns, err := corev1.NewNamespace(ctx, Namespace, &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.String(Namespace),
		},
	}, pulumi.Provider(k8s))
	if err != nil {
		return nil, err
	}

	chart, err := helmv3.NewChart(ctx, "netdata", helmv3.ChartArgs{
		Chart:     pulumi.String(HelmChart),
		Namespace: ns.Metadata.Name().Elem(),
		FetchArgs: helmv3.FetchArgs{
			Repo: pulumi.String(HelmRepoURL),
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
	}, pulumi.Provider(k8s), pulumi.DependsOn([]pulumi.Resource{ns}))
	if err != nil {
		return nil, err
	}

	chart.GetResource("v1/Service", "netdata", Namespace).ApplyT(func(r any) (*corev1.Service, error) {
		svc := r.(*corev1.Service)
		_, err = traefik.RegisterTailscaleSvc(ctx, "netdata", svc, args.TailscaleChart)
		if err != nil {
			return nil, err
		}
		return svc, nil
	})

	return &Deployment{
		Namespace: ns,
		Chart:     chart,
	}, nil
}
