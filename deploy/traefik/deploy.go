package traefik

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"homelab/tailscale"
)

//go:generate go run github.com/pulumi/crd2pulumi --go --goName "crds" "https://raw.githubusercontent.com/traefik/traefik/v3.1/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml"

var (
	// Namespace represents the K8S namespace in which the Traefik Helm release is deployed
	Namespace = "traefik-system"

	// LabelSelector represents the unique label key-value pair that the traefik Pod have to
	// be referenced by K8S services. Use the following form with Pulumi.
	//
	//     pulumi.ToStringMap(LabelSelector)
	//
	LabelSelector = map[string]string{
		"homelab/app": "traefik",
	}

	// WebPort is the port number that the traefik deployment listens on the HTTP protocol.
	WebPort = 8000

	// WebsecurePort is the port number that the traefik deployment listens on the HTTPs protocol.
	WebsecurePort = 8443

	// TraefikPort is the port number that the traefik deployment listens on for internal usage.
	TraefikPort = 9000

	// ChartCtxKey represent the context key to store the *helmv3.Release for the Traefik Chart to inject it later
	// pulumi resource dependency for other resources.
	ChartCtxKey = "traefikChart"
)

// DeployArgs is a struct that passes the arguments requiered to deploy the Traefik reverse proxy to the target K8S cluster
type DeployArgs struct{}

// Deployment is the result of the Deploy funtion providing references to the pulumi resources, so that they can be
// referenced throughout the whole pulumi deployment.
type Deployment struct {
	// Chart
	Chart *helmv3.Release

	// TailscaleServiceUI
	TailscaleServiceUI *corev1.Service
}

var (
	helmChart     = "traefik"
	helmRepoURL   = "https://traefik.github.io/charts/"
	disabledValue = map[string]any{"enabled": false}
)

// Deploy applies the Traefik Helm chart to the given K8S Cluster.
func Deploy(ctx *pulumi.Context, k8s *kubernetes.Provider, args *DeployArgs) (*Deployment, error) {
	tsChart := ctx.Value(tailscale.ChartCtxKey).(*helmv3.Release)

	rel, err := helmv3.NewRelease(ctx, "traefik", &helmv3.ReleaseArgs{
		Chart:           pulumi.String(helmChart),
		Namespace:       pulumi.String(Namespace),
		CreateNamespace: pulumi.Bool(true),
		Atomic:          pulumi.Bool(true),
		RepositoryOpts: &helmv3.RepositoryOptsArgs{
			Repo: pulumi.String(helmRepoURL),
		},
		Values: pulumi.ToMap(map[string]any{
			"ingressClass": disabledValue, // Disable Traefik IngressClass, only CRDs are used.
			"service":      disabledValue, // Disable the service in front of traefik, traefik is used as en entrypoint but only via Tailscale services.
			"providers": map[string]any{
				"kubernetesIngress": disabledValue, // Disable Ingress service discovery as the IngressClass is also disabled.
				"kubernetesCRD": map[string]any{
					"enabled":             true,
					"allowCrossNamespace": true,
				},
			},
			"globalArguments":   []string{"--global.sendAnonymousUsage=false"},
			"namespaceOverride": Namespace,
			"deployment": map[string]any{
				"enabled":   true,
				"kind":      "Deployment",
				"replicas":  1,
				"podLabels": LabelSelector,
			},
			"ports": map[string]any{
				"web": map[string]any{
					"port":     WebPort,
					"protocol": "TCP",
				},
				"websecure": map[string]any{
					"port":     WebsecurePort,
					"protocol": "TCP",
				},
				"traefik": map[string]any{
					"port":     TraefikPort,
					"protocol": "TCP",
				},
			},
			"ingressRoute": map[string]any{
				"ping": disabledValue,
				"dashboard": map[string]any{
					"enabled": true,
				},
			},
		}),
	}, pulumi.Provider(k8s))
	if err != nil {
		return nil, err
	}

	svc, err := corev1.NewService(ctx, "traefik-ui-svc", &corev1.ServiceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: rel.Status.Namespace(),
			Annotations: pulumi.ToStringMap(map[string]string{
				"tailscale.com/hostname": "traefik",
			}),
		},
		Spec: &corev1.ServiceSpecArgs{
			Type:              pulumi.String("LoadBalancer"),
			LoadBalancerClass: pulumi.String("tailscale"),
			Selector:          pulumi.ToStringMap(LabelSelector),
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port:       pulumi.Int(80),
					TargetPort: pulumi.Int(TraefikPort),
				},
			},
		},
	}, pulumi.Provider(k8s), pulumi.DependsOn([]pulumi.Resource{tsChart}))
	if err != nil {
		return nil, err
	}

	return &Deployment{
		Chart:              rel,
		TailscaleServiceUI: svc,
	}, nil
}
