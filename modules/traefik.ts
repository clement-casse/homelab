import * as k8s from "@pulumi/kubernetes";
import * as pulumi from "@pulumi/pulumi";
import { TailscaleOperator } from "./tailscale";
import { Middleware, IngressRoute } from "../crds/traefik/traefik/v1alpha1";

export interface TraefikArgs {
    namespace: string,
    tailscaleOperator: TailscaleOperator,
}

export class Traefik extends pulumi.ComponentResource {
    public readonly namespace: k8s.core.v1.Namespace;
    public readonly helmRelease: k8s.helm.v3.Release;
    public readonly traefikUiService: k8s.core.v1.Service;

    readonly tailscaleOperator: TailscaleOperator;
    readonly deployLabels: Record<string, string>;

    public static readonly HELM_CHART_REPO = "https://traefik.github.io/charts/";
    public static readonly HELM_CHART_NAME = "traefik";

    static readonly PORT_WEB = 8000;
    static readonly PORT_WEBSECURE = 8443;
    static readonly PORT_INTERNAL = 9000;

    constructor(name: string, args: TraefikArgs, opts: pulumi.ComponentResourceOptions = {}) {
        super("raz_algethi:homelab:Traefik", name, {}, opts);

        this.deployLabels = { "homelab/app": name };

        this.tailscaleOperator = args.tailscaleOperator;

        const helmValues = {
            "ingressClass": { "enabled": false }, // Disable Traefik IngressClass, only CRDs are used.
            "service": { "enabled": false }, // Disable the service in front of traefik, traefik is used as en entrypoint but only via Tailscale services.
            "providers": {
                "kubernetesIngress": { "enabled": false }, // Disable Ingress service discovery as the IngressClass is also disabled.
                "kubernetesCRD": {
                    "enabled": true,
                    "allowCrossNamespace": true,
                },
            },
            "globalArguments": ["--global.sendAnonymousUsage=false"],
            "namespaceOverride": args.namespace,
            "deployment": {
                "enabled": true,
                "kind": "Deployment",
                "replicas": 1,
                "podLabels": this.deployLabels,
            },
            "ports": {
                "web": { "port": Traefik.PORT_WEB, "protocol": "TCP" },
                "websecure": { "port": Traefik.PORT_WEBSECURE, "protocol": "TCP" },
                "traefik": { "port": Traefik.PORT_INTERNAL, "protocol": "TCP" },
            },
            "ingressRoute": {
                "ping": { "enabled": false },
                "dashboard": { "enabled": true },
            }
        };

        this.helmRelease = new k8s.helm.v3.Release(`${name}-chart`, {
            chart: Traefik.HELM_CHART_NAME,
            repositoryOpts: { repo: Traefik.HELM_CHART_REPO },
            namespace: args.namespace,
            createNamespace: true,
            atomic: true,
            values: helmValues,
        }, { parent: this });

        this.namespace = k8s.core.v1.Namespace.get(
            args.namespace,
            pulumi.interpolate`${this.helmRelease.status.namespace}`,
            { parent: this },
        );

        this.traefikUiService = new k8s.core.v1.Service(`${name}-ui-svc`, {
            metadata: {
                namespace: this.namespace.metadata.name,
                annotations: { "tailscale.com/hostname": name },
            },
            spec: {
                type: "LoadBalancer",
                loadBalancerClass: "tailscale",
                selector: this.deployLabels,
                ports: [
                    { port: 80, targetPort: Traefik.PORT_INTERNAL },
                ],
            }
        }, { parent: this });

        this.registerOutputs();
    }

    registerWebServiceInTailscale(name: string, svc: k8s.core.v1.Service, port: string | number) {
        const fqdn = `${name}.${this.tailscaleOperator.tailnet}.ts.net`;

        new k8s.core.v1.Service(`${name}-ts-lb-svc`, {
            metadata: {
                namespace: this.namespace.metadata.name,
                annotations: { "tailscale.com/hostname": name },
            },
            spec: {
                type: "LoadBalancer",
                loadBalancerClass: "tailscale",
                selector: this.deployLabels,
                ports: [
                    { name: "http-web", port: 80, targetPort: Traefik.PORT_WEB },
                    { name: "http-websecure", port: 443, targetPort: Traefik.PORT_WEBSECURE },
                ],
            }
        }, { parent: this.tailscaleOperator, dependsOn: [this.helmRelease, svc] });

        // Creates a middleware in traefik that redirects the requests which Host is either the
        // tailscale address or short name to the fully qualified address.
        const redirectMiddleware = new Middleware(`${name}-redirect-mw`, {
            metadata: { namespace: this.namespace.metadata.name },
            spec: {
                redirectRegex: {
                    permanent: true,
                    regex: `^https?://(?:100(?:\\.[0-9]{1,3}){3}|${name})(/.*)`,
                    replacement: `http://${fqdn}\${1}`,
                }
            }
        }, { parent: this, dependsOn: [this.helmRelease, svc] });

        new IngressRoute(`${name}-ts-ingressroute`, {
            metadata: { namespace: this.namespace.metadata.name },
            spec: {
                entryPoints: ["web", "websecure"],
                routes: [{
                    kind: "Rule",
                    match: "Host(`" + fqdn + "`)",
                    services: [{
                        kind: "Service",
                        name: svc.metadata.name,
                        namespace: svc.metadata.namespace,
                        port: port,
                    }],
                }, {
                    kind: "Rule",
                    match: "HostRegexp(`^(100(\\.[0-9]{1,3}){3}|" + name + ")$`)",
                    middlewares: [{
                        name: redirectMiddleware.metadata.name,
                        namespace: redirectMiddleware.metadata.namespace,
                    }],
                    services: [{ kind: "TraefikService", name: "noop@internal" }],
                }],
            }
        }, { parent: this, dependsOn: [this.helmRelease, svc] });
    }

}