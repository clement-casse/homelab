import * as k8s from "@pulumi/kubernetes";
import * as pulumi from "@pulumi/pulumi";

export interface CsiDriverSmbArgs {
    namespace: string,
}

export class CsiDriverSmb extends pulumi.ComponentResource {
    public readonly helmRelease: k8s.helm.v3.Release;
    
    public static readonly HELM_CHART_REPO = "https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/charts";
    public static readonly HELM_CHART_NAME = "csi-driver-smb";

    readonly namespace: k8s.core.v1.Namespace;

    static readonly CREDENTIALS_USERNAME_KEY = "username";
    static readonly CREDENTIALS_PASSWORD_KEY = "password";

    constructor(name: string, args: CsiDriverSmbArgs, opts: pulumi.ComponentResourceOptions = {}) {
        super("raz_algethi:homelab:CsiDriverSmb", name, {}, opts);

        this.helmRelease = new k8s.helm.v3.Release(`${name}-chart`, {
            chart: CsiDriverSmb.HELM_CHART_NAME,
            repositoryOpts: { repo: CsiDriverSmb.HELM_CHART_REPO },
            namespace: args.namespace,
            atomic: true,
        }, { parent: this });

        this.namespace = k8s.core.v1.Namespace.get(
            args.namespace,
            pulumi.interpolate`${this.helmRelease.status.namespace}`,
            { parent: this },
        );

        this.registerOutputs();
    }

    createStorageClass(name: string, addr: string, user: string, password: pulumi.Output<string>): k8s.storage.v1.StorageClass {
        const smbCredentials = new k8s.core.v1.Secret(`${name}-creds`, {
            metadata: { namespace: this.namespace.metadata.name },
            stringData: {
                [CsiDriverSmb.CREDENTIALS_USERNAME_KEY]: user,
                [CsiDriverSmb.CREDENTIALS_PASSWORD_KEY]: password,
            },
        }, { parent: this, dependsOn: [this.namespace] });

        return new k8s.storage.v1.StorageClass(`${name}`, {
            metadata: { name: name },
            provisioner: "smb.csi.k8s.io",
            reclaimPolicy: "Delete",
            volumeBindingMode: "Immediate",
            mountOptions: [
                "dir_mode=0777", "file_mode=0777",
                "uid=1001", "gid=1001",
                "noperm",
                "mfsymlinks",
                "cache=strict",
                "noserverino",
            ],
            parameters: {
                "source": addr,
                "subDir": "volumes/${pvc.metadata.name}",
                "onDelete": "retain",
                "csi.storage.k8s.io/provisioner-secret-name": smbCredentials.metadata.name,
                "csi.storage.k8s.io/provisioner-secret-namespace": smbCredentials.metadata.namespace,
                "csi.storage.k8s.io/node-stage-secret-name": smbCredentials.metadata.name,
                "csi.storage.k8s.io/node-stage-secret-namespace": smbCredentials.metadata.namespace,
            },
        }, { parent: this, dependsOn: [smbCredentials] });
    }
}