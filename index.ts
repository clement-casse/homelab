import * as pulumi from "@pulumi/pulumi";
import * as k8s from "@pulumi/kubernetes";

import { TailscaleOperator } from "./modules/tailscale";
import { Traefik } from "./modules/traefik";
import { CsiDriverSmb } from "./modules/smb";
import { Longhorn } from "./modules/longhorn";

const config = new pulumi.Config();

const k8sProvider = new k8s.Provider("cluster", {
    kubeconfig: config.get("kubernetes:kubeconfig")
});

const ts = new TailscaleOperator("tailscale-operator", {
    namespace: "tailscale-system",
    clientID: config.require("tailscaleK8SOperatorClientID"),
    oauthKey: config.requireSecret("tailscaleK8SOperatorSecret"),
}, { provider: k8sProvider });

const csiDriverSmb = new CsiDriverSmb("csi-driver-smb", { 
    namespace: "kube-system",
}, { provider: k8sProvider });

const smbStorageClass = csiDriverSmb.createStorageClass(
    "smb",
    config.require("smbserverAddress"),
    config.require("smbserverUsername"),
    config.requireSecret("smbserverPassword"),
);

const longhorn = new Longhorn("longhorn", { 
    namespace: "longhorn-system",
});

const reverseProxy = new Traefik("traefik", {
    namespace: "traefik-system",
}, { provider: k8sProvider });

reverseProxy.registerWebServiceInTailscale("longhorn", config.require("tailscaleTailnet"), longhorn.frontendService);