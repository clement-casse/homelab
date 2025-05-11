import * as k8s from "@pulumi/kubernetes";
import * as pulumi from "@pulumi/pulumi";

export interface LonghornArgs {
    namespace: string,
}

export class Longhorn extends pulumi.ComponentResource {
    public readonly namespace: k8s.core.v1.Namespace;
    public readonly helmRelease: k8s.helm.v3.Release;
    public readonly storageClass: k8s.storage.v1.StorageClass;
    public readonly frontendService: k8s.core.v1.Service;

    public static readonly HELM_CHART_REPO = "https://charts.longhorn.io";
    public static readonly HELM_CHART_NAME = "longhorn";

    constructor(name: string, args: LonghornArgs, opts: pulumi.ComponentResourceOptions = {}) {
        super("raz_algethi:homelab:Longhorn", name, {}, opts);

        const helmValues = {
            "service": {
                "ui": { "type": "ClusterIP" },
            },
            "longhornUI": { "replicas": 1 },
            "defaultSettings": {
                "deletingConfirmationFlag": "true",
            },
        };

        this.helmRelease = new k8s.helm.v3.Release(`${name}-chart`, {
            chart: Longhorn.HELM_CHART_NAME,
            repositoryOpts: { repo: Longhorn.HELM_CHART_REPO },
            namespace: args.namespace,
            createNamespace: true,
            atomic: true,
            values: helmValues,
        }, { parent: this });

        this.namespace = k8s.core.v1.Namespace.get(
            args.namespace,
            pulumi.interpolate`${this.helmRelease.status.namespace}`,
            { parent: this, dependsOn: [this.helmRelease] },
        );

        this.storageClass = k8s.storage.v1.StorageClass.get(
            `${name}-storage-class`,
            pulumi.interpolate`${this.helmRelease.status.apply(_ => "longhorn")}`,
            { parent: this, dependsOn: [this.helmRelease] },
        );

        this.frontendService = k8s.core.v1.Service.get(
            `${name}-frontend-svc`,
            pulumi.interpolate`${this.helmRelease.status.namespace}/${name}-frontend`,
            { parent: this, dependsOn: [this.helmRelease] },
        );

        this.registerOutputs();
    }
}