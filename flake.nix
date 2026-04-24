{
  description = "muxrun development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          version = "0.11.4";
        in
        {
          default = pkgs.buildGoModule {
            pname = "muxrun";
            inherit version;
            src = ./.;
            vendorHash = "sha256-LQuko0KeIZoIb1rmd+GyNUIUTB1yTP/BGMd7kgu39/0=";
            ldflags = [ "-X github.com/tkuramot/muxrun/cmd.version=${version}" ];

            nativeBuildInputs = [ pkgs.makeWrapper ];

            postInstall = ''
              wrapProgram $out/bin/muxrun \
                --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.tmux ]}
            '';

            meta = with pkgs.lib; {
              description = "A CLI tool that launches and manages multiple applications in groups using tmux";
              homepage = "https://github.com/tkuramot/muxrun";
              license = licenses.mit;
              mainProgram = "muxrun";
            };
          };
        }
      );

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
            ];
          };
        }
      );
    };
}
