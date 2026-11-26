{
  description = "painel: a terminal home screen for tools, scripts, projects and links";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = {
    self,
    nixpkgs,
  }: let
    systems = ["x86_64-linux" "aarch64-linux"];
    forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
  in {
    packages = forAllSystems (pkgs: {
      default = pkgs.buildGoModule {
        pname = "painel";
        version = "0.1.0";
        src = self;
        vendorHash = "sha256-xTPH5pwh4Reqnj6u4GuVyHDVQh5uoPeW+hUwY7aVdzA=";
        env.CGO_ENABLED = 0;
        ldflags = ["-s" "-w" "-X main.version=0.1.0"];
        meta = {
          description = "A terminal home screen for tools, scripts, projects and links";
          mainProgram = "painel";
        };
      };
    });

    devShells = forAllSystems (pkgs: {
      default = pkgs.mkShell {
        packages = [pkgs.go pkgs.gopls pkgs.gotools];
        env.CGO_ENABLED = 0;
      };
    });

    # `nix flake check` builds the package, which runs `go test`.
    checks = forAllSystems (pkgs: {
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
    });

    formatter = forAllSystems (pkgs: pkgs.alejandra);

    homeManagerModules.default = import ./nix/home-manager.nix self;
  };
}
