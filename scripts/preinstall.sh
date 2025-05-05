#!/bin/bash

crd2pulumi --nodejs --nodejsPath=./crds/traefik --force 'https://raw.githubusercontent.com/traefik/traefik/v3.1/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml'

crd2pulumi --nodejs --nodejsPath=./crds/tailscale --force 'https://raw.githubusercontent.com/tailscale/tailscale/refs/heads/main/cmd/k8s-operator/deploy/crds/tailscale.com_connectors.yaml' \
    'https://raw.githubusercontent.com/tailscale/tailscale/refs/heads/main/cmd/k8s-operator/deploy/crds/tailscale.com_dnsconfigs.yaml' \
    'https://raw.githubusercontent.com/tailscale/tailscale/refs/heads/main/cmd/k8s-operator/deploy/crds/tailscale.com_proxyclasses.yaml' \
    'https://raw.githubusercontent.com/tailscale/tailscale/refs/heads/main/cmd/k8s-operator/deploy/crds/tailscale.com_proxygroups.yaml' \
    'https://raw.githubusercontent.com/tailscale/tailscale/refs/heads/main/cmd/k8s-operator/deploy/crds/tailscale.com_recorders.yaml'
