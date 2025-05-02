# Homelab

This repository hosts the code of my homelab setup running on a self-hosted Kubernetes Cluster.

## Repository Structure

- **[`cloud-configs/`](./cloud-configs/)** stores templates for [cloud-init config files](https://cloudinit.readthedocs.io/en/latest/reference/examples.html) that install and configure **K3S** and **Tailscale**.
  Despite some general recommendations - _and common sense too_ - I still decided to go for a templating approach for Cloud-config files.
  However the approach was to keep templated values rather minimal and only confined to the injection of simple config and secrets into the YAML file.
  I took this approach to avoid leaking my Tailscale OAuth Keys in a first place.

- **[`deploy/`](./deploy/)** defines a Pulumi Project that deploys on the K3S cluster all the services of the homelab.
  The Pulumi project is written in Go (which is one of the design choices I regret the most because it leads to some ill designed code, _imho_).
  Most of the configuration is stored in the [Pulumi ESC](https://www.pulumi.com/product/esc/) app.
  It uses an environment which is expected to hold the following values, the one marked as `"[secret]"` are expected to be defined as [ESC Secrets](https://www.pulumi.com/docs/esc/get-started/store-and-retrieve-secrets/):
  
  ```yaml
  values:
      pulumiConfig:
          # The KUBECONFIG for the target K3S Cluster
          kubernetes:kubeconfig: "[secret]"
          # Configuration for the Tailscale Operator:
          tailscaleTailnet: ""                     # The name of the Tailnet
          tailscaleK8SOperatorClientID: "[secret]" # The ClientID of the Tailscale OAuth Client
          tailscaleK8SOperatorSecret: "[secret]"   # The Token of the Tailscale OAuth Client
          # Configuration for the SMB Storage class in K8S:
          smbserverAddress: ""          # The address of the SMB server
          smbserverUsername: ""         # The username to log in the SMB server
          smbserverPassword: "[secret]" # The password to log in the SMB server
  ```

- **[`makeDevEnv.sh`](./makeDevEnv.sh)** is a shell script that spins a K3S cluster on the local machine with the cloud configs defined in [`cloud-configs/`](./cloud-configs/).
