package traefik

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"homelab/tailscale"
	traefikv1alpha1 "homelab/traefik/kubernetes/traefik/v1alpha1"
)

type TailscaleSvc struct {
	Service      *corev1.Service
	IngressRoute *traefikv1alpha1.IngressRoute
}

// RegisterTailscaleSvc registers a Tailscale Service behind the Traefik Reverse Proxy
func RegisterTailscaleSvc(ctx *pulumi.Context, tsName string, svc *corev1.Service) (*TailscaleSvc, error) {
	tailnet := config.New(ctx, "").Get(tailscale.TailnetESCConfigKey)
	tsChart := ctx.Value(tailscale.ChartCtxKey).(*helmv3.Release)
	traefikChart := ctx.Value(ChartCtxKey).(*helmv3.Release)

	// Create a K8S service of type LoadBalancer that gets its IP from Tailscale and link it to the Traefik Deployment.
	tSvc, err := corev1.NewService(ctx, fmt.Sprintf("%s-svc", tsName), &corev1.ServiceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: pulumi.String(Namespace),
			Annotations: pulumi.ToStringMap(map[string]string{
				"tailscale.com/hostname": tsName,
			}),
		},
		Spec: &corev1.ServiceSpecArgs{
			Type:              pulumi.String("LoadBalancer"),
			LoadBalancerClass: pulumi.String("tailscale"),
			Selector:          pulumi.ToStringMap(LabelSelector),
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Name:       pulumi.String("http-web"),
					Port:       pulumi.Int(80),
					TargetPort: pulumi.Int(WebPort),
				},
				&corev1.ServicePortArgs{
					Name:       pulumi.String("http-websecure"),
					Port:       pulumi.Int(443),
					TargetPort: pulumi.Int(WebsecurePort),
				},
			},
		},
	}, pulumi.DependsOn([]pulumi.Resource{tsChart}))
	if err != nil {
		return nil, err
	}

	// Create a Traefik IngressRoute Custom resource that define the rule for Traefik to route HTTP Requests from that Tailscale host
	// to the given K8S service.
	ir, err := traefikv1alpha1.NewIngressRoute(ctx,
		fmt.Sprintf("%s-ingressroute", tsName),
		&traefikv1alpha1.IngressRouteArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: pulumi.String(Namespace),
			},
			Spec: traefikv1alpha1.IngressRouteSpecArgs{
				EntryPoints: pulumi.ToStringArray([]string{"web", "websecure"}),
				Routes: traefikv1alpha1.IngressRouteSpecRoutesArray{
					traefikv1alpha1.IngressRouteSpecRoutesArgs{
						Kind:  pulumi.String("Rule"),
						Match: pulumi.String(fmt.Sprintf("Host(`%s.%s.ts.net`)", tsName, tailnet)),
						Services: traefikv1alpha1.IngressRouteSpecRoutesServicesArray{
							traefikv1alpha1.IngressRouteSpecRoutesServicesArgs{
								Kind:      pulumi.String("Service"),
								Name:      svc.Metadata.Name().Elem(),
								Namespace: svc.Metadata.Namespace(),
								Port:      svc.Spec.Ports().Index(pulumi.Int(0)).Port(),
							},
						},
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{svc, traefikChart}))
	if err != nil {
		return nil, err
	}

	return &TailscaleSvc{
		Service:      tSvc,
		IngressRoute: ir,
	}, nil
}
