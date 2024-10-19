{
  description = "Nix flake for testing and deploying my homelab";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, utils }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };

        nativeBuildInputs = with pkgs; [
          kubectl
          kubernetes-helm
          kustomize
          kubeconform
          gettext
          yq-go

          pulumi-bin
          pulumi-esc

          go
          gopls
          gotools
          go-tools
        ];
      in
      with pkgs;
      {
        formatter = nixpkgs-fmt;

        devShells.default = mkShell {
          inherit nativeBuildInputs;
          # Load ESC environmentVariables in the devShell
          shellHook = ''
            eval $(esc open raz_algethi/default/homelab-dev --format shell);
          '';
        };
      }
    );
}
