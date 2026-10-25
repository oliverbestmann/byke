{
  description = "byke development and snapshot test shells";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "aarch64-linux"
        "x86_64-linux"
      ];

      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      # libraries needed to build byke (glfw, webgpu, oto) with cgo
      libraries =
        pkgs: with pkgs; [
          alsa-lib
          libGL
          libglvnd
          libxkbcommon
          vulkan-headers
          vulkan-loader
          wayland
          libx11
          libxcursor
          libxext
          libxi
          libxinerama
          libxrandr
          libxxf86vm
        ];

      shell =
        pkgs: extra:
        pkgs.mkShell (
          {
            name = "byke";

            buildInputs = libraries pkgs;
            packages = [
              pkgs.go
              pkgs.pkg-config
            ];

            # fixes some compile issue with cgo
            hardeningDisable = [ "fortify" ];

            LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath (libraries pkgs);
          }
          // extra
        );
    in
    {
      devShells = forAllSystems (pkgs: {
        default = shell pkgs { };

        # renders with mesa's software vulkan driver (lavapipe) from this flake,
        # so snapshot hashes are the same locally and in CI
        test = shell pkgs {
          VK_DRIVER_FILES = "${pkgs.mesa}/share/vulkan/icd.d/lvp_icd.${pkgs.stdenv.hostPlatform.parsed.cpu.name}.json";
        };
      });
    };
}
