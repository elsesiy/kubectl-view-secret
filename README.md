# kubectl-view-secret

[![Go Report Card](https://goreportcard.com/badge/github.com/elsesiy/kubectl-view-secret)](https://goreportcard.com/report/github.com/elsesiy/kubectl-view-secret)
![CI](https://github.com/elsesiy/kubectl-view-secret/actions/workflows/ci.yml/badge.svg)
[![codecov](https://codecov.io/github/elsesiy/kubectl-view-secret/graph/badge.svg?token=RODJX5GLDB)](https://codecov.io/github/elsesiy/kubectl-view-secret)
[![Twitter](https://img.shields.io/badge/twitter-@elsesiy-blue.svg)](http://twitter.com/elsesiy)
[![GitHub release](https://img.shields.io/github/release/elsesiy/kubectl-view-secret.svg)](https://github.com/elsesiy/kubectl-view-secret/releases)

![gif](./media/view-secret.gif)

This plugin allows for easy secret decoding. Useful if you want to see what's inside of a secret without always go through the following:

1. `kubectl get secret <secret> -o yaml`
2. Copy base64 encoded secret
3. `echo "b64string" | base64 -d`

Instead you can now do:

    # print secret keys
    kubectl view-secret <secret>

    # decode specific entry
    kubectl view-secret <secret> <key>

    # decode all contents
    kubectl view-secret <secret> -a/--all

    # print keys for secret in different namespace
    kubectl view-secret <secret> -n/--namespace <ns>

    # print keys for secret in different context
    kubectl view-secret <secret> -c/--context <ctx>

    # print keys for secret by providing kubeconfig
    kubectl view-secret <secret> -k/--kubeconfig <cfg>

    # suppress info output
    kubectl view-secret <secret> -q/--quiet

## Usage

### Shell completion

To enable completion for this plugin, you will need to use kubectl 1.26 or
above. Then, create a script named `kubectl_complete-view_secret` with the
following content:

```bash
#!/bin/bash
kubectl view-secret __complete "$@"
```

and put it somewhere in your PATH.

### Krew

This plugin is available through [krew](https://krew.dev) via:

```sh
kubectl krew install view-secret
```

### Binary releases

#### GitHub

You can find the latest binaries in the [releases](https://github.com/elsesiy/kubectl-view-secret/releases) section.  
To install it, place it somewhere in your `$PATH` for `kubectl` to pick it up.

**Note**: If you build from source or download the binary, you'll have to change the name of the binary to `kubectl-view_secret` (`-` to `_` in `view-secret`)
due to the enforced naming convention for plugins by `kubectl`. More on this [here](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/#naming-a-plugin).

#### AUR package

You can find the latest package description for Arch users [here](https://aur.archlinux.org/packages/kubectl-view-secret-bin).

Contribution by [@jocelynthode](https://github.com/jocelynthode)

#### Nix
You can install the latest version from Nixpkgs ([24.05](https://search.nixos.org/packages?channel=24.05&from=0&size=50&sort=relevance&type=packages&query=kubectl-view-secret), [unstable](https://search.nixos.org/packages?channel=unstable&show=kubectl-view-secret&from=0&size=50&sort=relevance&type=packages&query=kubectl-view-secret)) or try it via a temporary nix-shell:

```
nix-shell -p kubectl-view-secret
```

### Build from source

    # Clone this repository (or your fork)
    git clone https://github.com/elsesiy/kubectl-view-secret
    cd kubectl-view-secret
    make

## License

This repository is available under the [MIT license](https://choosealicense.com/licenses/mit/).
