#!/bin/bash

crd2pulumi --nodejs --nodejsPath=./crdstraefik --force 'https://raw.githubusercontent.com/traefik/traefik/v3.1/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml'

crd2pulumi --nodejs --nodejsPath=./crdstailscale --force 'https://raw.githubusercontent.com/tailscale/tailscale/refs/heads/main/cmd/k8s-operator/deploy/crds/tailscale.com_connectors.yaml'
crd2pulumi --nodejs --nodejsPath=./crdstailscale --force 'https://raw.githubusercontent.com/tailscale/tailscale/refs/heads/main/cmd/k8s-operator/deploy/crds/tailscale.com_dnsconfigs.yaml'
crd2pulumi --nodejs --nodejsPath=./crdstailscale --force 'https://raw.githubusercontent.com/tailscale/tailscale/refs/heads/main/cmd/k8s-operator/deploy/crds/tailscale.com_proxyclasses.yaml'
crd2pulumi --nodejs --nodejsPath=./crdstailscale --force 'https://raw.githubusercontent.com/tailscale/tailscale/refs/heads/main/cmd/k8s-operator/deploy/crds/tailscale.com_proxygroups.yaml'
crd2pulumi --nodejs --nodejsPath=./crdstailscale --force 'https://raw.githubusercontent.com/tailscale/tailscale/refs/heads/main/cmd/k8s-operator/deploy/crds/tailscale.com_recorders.yaml'
