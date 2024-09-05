package smb

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	storagev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/storage/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	// Namespace represents the K8S namespace in which the SMB Driver is deployed
	Namespace = "kube-system"

	// AddressESCConfigKey is key of the pulumiconfig in the environment that stores the address of the SMB server
	AddressESCConfigKey = "smbserverAddress"

	// UsernameESCConfigtKey is key of the pulumiconfig in the environment that stores the username of the user to log in the SMB server
	UsernameESCConfigtKey = "smbserverUsername"

	// PasswordESCSecretKey is key of the pulumiconfig in the environment that stores the password of the user to log in the SMB server
	PasswordESCSecretKey = "smbserverPassword"

	// StorageClassCtxKey represent the context key to store the SMB storage class
	StorageClassCtxKey = "smbStorageClass"
)

// DeployArgs is a struct that passes the arguments requiered to deploy the SMB storage class in the target K8S cluster
type DeployArgs struct {
	Address  pulumi.String
	Username pulumi.String
	Password pulumi.StringOutput
}

// Deployment is the result of the Deploy funtion providing references to the pulumi resources, so that they can be
// referenced throughout the whole pulumi deployment.
type Deployment struct {
	// Chart
	Chart *helmv3.Release

	// StorageClass
	StorageClass *storagev1.StorageClass
}

var (
	helmChart   = "csi-driver-smb"
	helmRepoURL = "https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/charts"
)

// Deploy applies the CSI SMB Driver helm chart to the given K8S cluster. It also applies a Kubernetes secret for credentials
// and the defintion of a storage class for SMB persistent volumes.
func Deploy(ctx *pulumi.Context, k8s *kubernetes.Provider, args *DeployArgs) (*Deployment, error) {
	rel, err := helmv3.NewRelease(ctx, "csi-driver-smb", &helmv3.ReleaseArgs{
		Chart:     pulumi.String(helmChart),
		Namespace: pulumi.String(Namespace),
		Atomic:    pulumi.Bool(true),
		RepositoryOpts: helmv3.RepositoryOptsArgs{
			Repo: pulumi.String(helmRepoURL),
		},
	}, pulumi.Provider(k8s))
	if err != nil {
		return nil, err
	}

	credentials, err := corev1.NewSecret(ctx, "smbcreds", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: pulumi.String(Namespace),
		},
		StringData: pulumi.StringMap{
			"username": args.Username,
			"password": args.Password,
		},
	}, pulumi.Provider(k8s))
	if err != nil {
		return nil, err
	}

	storageClass, err := storagev1.NewStorageClass(ctx, "smb", &storagev1.StorageClassArgs{
		Provisioner:       pulumi.String("smb.csi.k8s.io"),
		ReclaimPolicy:     pulumi.String("Delete"),
		VolumeBindingMode: pulumi.String("Immediate"),
		MountOptions: pulumi.ToStringArray([]string{
			"dir_mode=0777", "file_mode=0777",
			"uid=1001", "gid=1001",
			"noperm",
			"mfsymlinks",
			"cache=strict",
			"noserverino",
		}),
		Parameters: pulumi.StringMap{
			"source":   args.Address,
			"subDir":   pulumi.String("volumes/${pvc.metadata.name}"),
			"onDelete": pulumi.String("retain"),

			"csi.storage.k8s.io/provisioner-secret-name":      credentials.Metadata.Name().Elem(),
			"csi.storage.k8s.io/provisioner-secret-namespace": credentials.Metadata.Namespace().Elem(),
			"csi.storage.k8s.io/node-stage-secret-name":       credentials.Metadata.Name().Elem(),
			"csi.storage.k8s.io/node-stage-secret-namespace":  credentials.Metadata.Namespace().Elem(),
		},
	}, pulumi.Provider(k8s), pulumi.DependsOn([]pulumi.Resource{rel}))
	if err != nil {
		return nil, err
	}

	return &Deployment{
		Chart:        rel,
		StorageClass: storageClass,
	}, nil
}
