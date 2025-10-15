{
  description = "OpenCode - Terminal-based AI assistant for software development";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    treefmt-nix.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = {
    self,
    nixpkgs,
    treefmt-nix,
    ...
  }: let
    supportedSystems = [
      "x86_64-linux"
      "x86_64-darwin"
      "aarch64-linux"
      "aarch64-darwin"
    ];
    forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
  in {
    packages = forAllSystems (system: let
      pkgs = import nixpkgs {
        inherit system;
      };
    in {
      default = pkgs.buildGo124Module {
        pname = "opencode";
        version = "0.1.0";
        src = self;
        vendorHash = "sha256-Kcwd8deHug7BPDzmbdFqEfoArpXJb1JtBKuk+drdohM=";
        doCheck = false;

        ldflags = ["-s" "-w"];

        meta = with pkgs.lib; {
          description = "OpenCode - Terminal-based AI assistant for software development";
          homepage = "https://github.com/opencode-ai/opencode";
          license = licenses.mit;
          mainProgram = "opencode";
        };
      };
    });

    devShells = forAllSystems (system: let
      pkgs = import nixpkgs {
        inherit system;
      };

      scripts = {
        gen = {
          exec = ''go generate ./...'';
          description = "Run code generation";
        };
        lint = {
          exec = ''golangci-lint run'';
          description = "Run Linting Steps for go files";
        };
        build = {
          exec = ''go build -o opencode .'';
          description = "Build the OpenCode CLI";
        };
        run = {
          exec = ''go run .'';
          description = "Run the OpenCode CLI";
        };
        tests = {
          exec = ''go test ./...'';
          description = "Run all go tests";
        };
      };

      scriptPackages =
        pkgs.lib.mapAttrs
        (
          name: script:
            pkgs.writeShellApplication {
              inherit name;
              text = script.exec;
              runtimeInputs = script.deps or [];
            }
        )
        scripts;

      buildWithSpecificGo = pkg: pkg.override {buildGoModule = pkgs.buildGo124Module;};
    in {
      default = pkgs.mkShell {
        name = "opencode-dev";

        packages = with pkgs;
          [
            # Nix tools
            alejandra
            nixd
            statix
            deadnix

            # Go Tools
            go_1_24
            air
            golangci-lint
            gopls
            (buildWithSpecificGo revive)
            (buildWithSpecificGo golines)
            (buildWithSpecificGo golangci-lint-langserver)
            (buildWithSpecificGo gomarkdoc)
            (buildWithSpecificGo gotests)
            (buildWithSpecificGo gotools)
            (buildWithSpecificGo reftools)
            pprof
            graphviz
            goreleaser
            cobra-cli

            # CLI development tools
            sqlite
          ]
          ++ builtins.attrValues scriptPackages;

        shellHook = ''
          export REPO_ROOT=$(git rev-parse --show-toplevel)
          echo "OpenCode development environment loaded"
          echo "Run 'build' to build the CLI"
          echo "Run 'run' to start the OpenCode CLI"
          echo "Run 'tests' to run all tests"
        '';
      };
    });

    # Runnable with: > nix fmt
    formatter = forAllSystems (system: let
      pkgs = nixpkgs.legacyPackages.${system};
      treefmtModule = {
        projectRootFile = "flake.nix";
        programs = {
          alejandra.enable = true; # Nix formatter
          gofmt.enable = true; # Go formatter
        };
      };
    in
      treefmt-nix.lib.mkWrapper pkgs treefmtModule);
  };
}
