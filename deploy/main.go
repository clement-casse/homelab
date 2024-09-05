package main

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"homelab/netdata"
	"homelab/postgres"
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

		smbDeploy, err := smb.Deploy(ctx, k8s, &smb.DeployArgs{
			Address:  pulumi.String(config.Get(smb.AddressESCConfigKey)),
			Username: pulumi.String(config.Get(smb.UsernameESCConfigtKey)),
			Password: config.GetSecret(smb.PasswordESCSecretKey),
		})
		if err != nil {
			return
		}
		ctx = ctx.WithValue(smb.StorageClassCtxKey, smbDeploy.StorageClass)

		tailscaleDeploy, err := tailscale.Deploy(ctx, k8s, &tailscale.DeployArgs{
			ClientID: pulumi.String(config.Get(tailscale.ClientIDESCConfigKey)),
			OAuthKey: config.GetSecret(tailscale.OAuthKeyESCSecretKey),
		})
		if err != nil {
			return
		}
		ctx = ctx.WithValue(tailscale.ChartCtxKey, tailscaleDeploy.Chart)

		traefikDeploy, err := traefik.Deploy(ctx, k8s, &traefik.DeployArgs{})
		if err != nil {
			return
		}
		ctx = ctx.WithValue(traefik.ChartCtxKey, traefikDeploy.Chart)

		if _, err = netdata.Deploy(ctx, k8s, &netdata.DeployArgs{
			Rooms: pulumi.String(config.Get(netdata.RoomsESCConfigKey)),
			Token: config.GetSecret(netdata.TokenESCSecretKey),
		}); err != nil {
			return
		}

		pgDeploy, err := postgres.Deploy(ctx, k8s, &postgres.DeployArgs{
			InstallUI: true,
		})
		if err != nil {
			return
		}
		ctx = ctx.WithValue(postgres.ChartCtxKey, pgDeploy.OperatorChart)

		_ = ctx

		return
	})
}
