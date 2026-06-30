{
  description = "tagfs - A semantic FUSE to tidy up your home";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }: {
    packages.x86_64-linux.default = nixpkgs.legacyPackages.x86_64-linux.buildGoModule {
      pname = "tagfs";
      version = "0.1.0";
      src = ./.;
      vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="; # 'go mod vendor'
      meta.mainProgram = "tagfs";
    };
    
    homeManagerModules.default = import ./module.nix self;
  };
}
