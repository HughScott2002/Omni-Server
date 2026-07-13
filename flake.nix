{
  description = "Omni-Server — polyglot microservices dev environment (Go + Python/FastAPI + Kafka/Redis)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        # go.mod requires go 1.24 (1-users, 3-wallet, 4-transactions);
        # 5-fraud-detection asks for 1.22. Go 1.25 satisfies both.
        # (nixpkgs-unstable no longer packages go_1_24.)
        go = pkgs.go_1_25;

        python = pkgs.python312;
      in
      {
        devShells.default = pkgs.mkShell {
          name = "omni-server";

          packages = with pkgs; [
            # --- Go microservices (1-users, 3-wallet, 4-transactions, 5-fraud-detection) ---
            go
            gopls
            gotools          # goimports, etc.
            go-tools         # staticcheck
            golangci-lint    # Go linter
            delve            # dlv debugger

            # --- Python / FastAPI notification service (2-notification) ---
            python
            poetry
            ruff             # Python linter/formatter

            # native libs for python-snappy (kafka message compression) + common wheels
            snappy
            zlib
            stdenv.cc.cc.lib

            # --- Git hooks (.pre-commit-config.yaml) ---
            pre-commit

            # --- Local orchestration (makefile / docker-compose.yaml) ---
            docker-compose

            # --- Handy CLIs for poking the stack ---
            redis            # redis-cli
            kcat             # kafkacat — inspect Kafka topics
            curl
            jq
            git
          ];

          # python-snappy and some other wheels compile against C libs at
          # `poetry install` time; expose them to the toolchain.
          env = {
            LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath [
              pkgs.snappy
              pkgs.zlib
              pkgs.stdenv.cc.cc.lib
            ];
            # keep Go module/build caches inside the repo checkout
            GOFLAGS = "-mod=mod";
          };

          shellHook = ''
            # Install the git pre-commit hook once (no-op if already present).
            if [ -d .git ] && [ ! -f .git/hooks/pre-commit ]; then
              pre-commit install >/dev/null 2>&1 || true
            fi

            echo ""
            echo "  Omni-Server dev shell"
            echo "  ---------------------"
            echo "  go        $(${go}/bin/go version | awk '{print $3}')"
            echo "  python    $(${python}/bin/python --version | awk '{print $2}')"
            echo "  poetry    $(poetry --version 2>/dev/null | awk '{print $3}' | tr -d ')')"
            echo ""
            echo "  Full stack  : make build   (see 'make help' for all targets)"
            echo "  Go service  : cd 1-users && go run ./src   (also 3-wallet, 4-transactions, 5-fraud-detection)"
            echo "  Notification: cd 2-notification && poetry install && poetry run uvicorn app.api.main:app --reload"
            echo "  Lint / fmt  : golangci-lint run ./...   |   ruff check .   |   ruff format ."
            echo ""
          '';
        };
      });
}
