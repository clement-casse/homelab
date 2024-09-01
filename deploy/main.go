package main

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"homelab/netdata"
	"homelab/smb"
	"homelab/tailscale"
	"homelab/traefik"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) (err error) {
		config := config.New(ctx, "")
		k8s, err := kubernetes.NewProvider(ctx, "k8s-provider", &kubernetes.ProviderArgs{
			Kubeconfig: pulumi.String(config.Get("kubernetes:kubeconfig")),
		})
		if err != nil {
			return
		}

		if _, err = smb.Deploy(ctx, k8s, &smb.DeployArgs{
			Address:  pulumi.String(config.Get(smb.AddressESCConfigKey)),
			Username: pulumi.String(config.Get(smb.UsernameESCConfigtKey)),
			Password: config.GetSecret(smb.PasswordESCSecretKey),
		}); err != nil {
			return
		}

		tailscaleDeploy, err := tailscale.Deploy(ctx, k8s, &tailscale.DeployArgs{
			ClientID: pulumi.String(config.Get(tailscale.ClientIDESCConfigKey)),
			OAuthKey: config.GetSecret(tailscale.OAuthKeyESCSecretKey),
		})
		if err != nil {
			return
		}

		_, err = traefik.Deploy(ctx, k8s, &traefik.DeployArgs{
			ServiceDeps: []pulumi.Resource{tailscaleDeploy.Chart},
		})
		if err != nil {
			return
		}

		if _, err = netdata.Deploy(ctx, k8s, &netdata.DeployArgs{
			Rooms:          pulumi.String(config.Get(netdata.RoomsESCConfigKey)),
			Token:          config.GetSecret(netdata.TokenESCSecretKey),
			TailscaleChart: tailscaleDeploy.Chart,
		}); err != nil {
			return
		}

		return
	})
}
