import * as k8s from "@pulumi/kubernetes";
import * as pulumi from "@pulumi/pulumi";
// import * as tailscale from "@pulumi/tailscale";

export interface TailscaleOperatorArgs {
    namespace: string,
    clientID: string,
    oauthKey: pulumi.Output<string>,
}

export class TailscaleOperator extends pulumi.ComponentResource {
    public readonly namespace: k8s.core.v1.Namespace;
    public readonly helmRelease: k8s.helm.v3.Release;

    public static readonly HELM_CHART_REPO = "https://pkgs.tailscale.com/helmcharts";
    public static readonly HELM_CHART_NAME = "tailscale-operator";

    constructor(name: string, args: TailscaleOperatorArgs, opts: pulumi.ComponentResourceOptions = {}) {
        super("raz_algethi:homelab:TailscaleOperator", name, {}, opts);

        const helmValues = {
            "installCRDs": true,
            "oauth": {
                "clientId": args.clientID,
                "clientSecret": args.oauthKey,
            },
            "operatorConfig": {
                "defaultTags": ["tag:k8s-operator"],
            },
        };

        this.helmRelease = new k8s.helm.v3.Release(`${name}-chart`, {
            chart: TailscaleOperator.HELM_CHART_NAME,
            repositoryOpts: { repo: TailscaleOperator.HELM_CHART_REPO },
            namespace: args.namespace,
            createNamespace: true,
            atomic: true,
            values: helmValues,
        }, { parent: this });

        this.namespace = k8s.core.v1.Namespace.get(
            args.namespace,
            pulumi.interpolate `${this.helmRelease.status.namespace}`,
            { parent: this },
        );

        this.registerOutputs();
    }
}
