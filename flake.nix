{
  description = "Nix flake for creating a custom OpenTelemetry Collector";

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
