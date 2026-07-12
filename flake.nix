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
      vendorHash = "sha256-PLD7BnqA6ePJ39la7UuhfEPPa++fArIqpHT5a29tkBg=";
      meta.mainProgram = "tagfs";
    };
    
    homeManagerModules.default = import ./module.nix self;
  };
}
