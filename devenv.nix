{
  pkgs,
  ...
}:

{

  packages = [
    pkgs.git
    pkgs.go_1_25
  ];

  enterShell = ''
    git --version
    go version
  '';

  enterTest = ''
    echo "Running tests"
    go test -race ./...
  '';
}
