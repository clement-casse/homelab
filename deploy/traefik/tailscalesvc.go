package traefik

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"homelab/tailscale"
	traefikv1alpha1 "homelab/traefik/kubernetes/traefik/v1alpha1"
)

type TailscaleSvc struct {
	svc          *corev1.Service
	ingressRoute *traefikv1alpha1.IngressRoute
}

// RegisterTailscaleSvc registers a Tailscale Service behind the Traefik Reverse Proxy
func RegisterTailscaleSvc(ctx *pulumi.Context, tsName string, svc *corev1.Service, tsChart *helmv3.Release) (*TailscaleSvc, error) {
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
						Match: pulumi.String(fmt.Sprintf("Host(`%s.%s.ts.net`)", tsName, tailscale.TailnetESCConfigKey)),
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
		}, pulumi.DependsOn([]pulumi.Resource{svc}))
	if err != nil {
		return nil, err
	}

	return &TailscaleSvc{
		svc:          tSvc,
		ingressRoute: ir,
	}, nil
}
