{ pkgs ? import <nixpkgs> {} }:
  let
    lib = pkgs.lib;
  in
    pkgs.buildGoModule rec {
      name = "opencode";
      version = "0.0.34";

      src = pkgs.fetchFromGitHub {
        owner = "opencode-ai";
        repo = "opencode";
        tag = "v${version}";
        hash = "sha256-EaspkL0TEBJEUU3f75EhZ4BOIvbneUKnTNeNGhJdjYE=";
      };  

      vendorHash = "sha256-cFzkMunPkGQDFhQ4NQZixc5z7JCGNI7eXBn826rWEvk=";

      checkFlags =
      let
        skippedTests = [
          # permission denied
          "TestBashTool_Run"
          "TestSourcegraphTool_Run"
          "TestLsTool_Run"
        ];
      in
      [ "-skip=^${lib.concatStringsSep "$|^" skippedTests}$" ];

      meta = with lib; {
        description = "A powerful terminal-based AI assistant for developers, providing intelligent coding assistance directly in your terminal.";
        homepage = "https://github.com/opencode-ai/opencode";
        mainProgram = "opencode";
        license = licenses.mit;
      };
    }
