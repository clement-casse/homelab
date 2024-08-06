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

          #qemu
          qemu-utils

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
        };
      }
    );
}
