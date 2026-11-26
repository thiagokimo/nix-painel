# Home Manager module: installs painel and writes its config from Nix.
#
#   programs.painel = {
#     enable = true;
#     settings.scripts = [{ name = "sysup"; alias = "su"; cmd = ["sysup"]; pause = true; }];
#   };
self: {
  config,
  lib,
  pkgs,
  ...
}: let
  cfg = config.programs.painel;
  json = pkgs.formats.json {};
in {
  options.programs.painel = {
    enable = lib.mkEnableOption "painel, a terminal home screen";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      defaultText = lib.literalExpression "painel.packages.\${system}.default";
      description = "The painel package.";
    };

    settings = lib.mkOption {
      inherit (json) type;
      default = {};
      description = ''
        Written to $XDG_CONFIG_HOME/painel/config.json. See the README for the
        keys; anything left out keeps painel's default.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [cfg.package];
    xdg.configFile."painel/config.json" = lib.mkIf (cfg.settings != {}) {
      source = json.generate "painel-config.json" cfg.settings;
    };
  };
}
