{ config, lib, pkgs, ... }:

let
  cfg = config.programs.tagfs;
  configFile = pkgs.writeText "tagfs-config.json" (builtins.toJSON cfg.settings);
in {
  options.programs.tagfs = {
    enable = lib.mkEnableOption "tagfs";
    package = lib.mkPackageOption pkgs "tagfs" {};
    settings = lib.mkOption {
      type = lib.types.attrs;
      default = {};
      description = "Manage tagfs (sort inbox, confirm auto sorts, etc.)";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.user.services.tagfs = {
      Unit = { Description = "tagfs FUSE"; };
      Service = {
        ExecStart = "${cfg.package}/bin/tagfs mount --config ${configFile}";
        Restart = "on-failure";
      };
      Install = { WantedBy = [ "default.target" ]; };
      Environment = "TAGFS_SOCKET=/run/user/${builtins.toString config.ids.uids.user}/tagfs.sock";
    };
  };
}
