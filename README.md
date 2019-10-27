# kubectl-view-secret
[![Go Report Card](https://goreportcard.com/badge/github.com/elsesiy/kubectl-view-secret)](https://goreportcard.com/report/github.com/elsesiy/kubectl-view-secret)
[![Build Status](https://travis-ci.com/elsesiy/kubectl-view-secret.svg?branch=master)](https://travis-ci.com/elsesiy/kubectl-view-secret)
[![Twitter](https://img.shields.io/badge/twitter-@elsesiy-blue.svg)](http://twitter.com/elsesiy)
[![GitHub release](https://img.shields.io/github/release/elsesiy/kubectl-view-secret.svg)](https://github.com/elsesiy/kubectl-view-secret/releases)

This plugin allows for easy secret decoding. Useful if you want to see what's inside of a secret without always go throug the following:
1. `kubectl get secret <secret> -o yaml`
2. Copy base64 encoded secret
3. `echo "b64string" | base64 -d`

Instead you can now do:

    # print secret keys
    kubectl view-secret <secret>
    
    # decode specific entry
    kubectl view-secret <secret> <key>
    
    # decode all contents
    kubeclt view-secret <secret> -a/--all
    
    # print keys for secret in different namespace
    kubectl view-secret <secret> -n/--namespace <ns>
    
    # suppress info output
    kubectl view-secret <secret> -q/--quiet

## Build

    # Clone this repository (or your fork)
    git clone https://github.com/elsesiy/kubectl-view-secret
    cd kubectl-view-secret
    make

## License

This repository is available under the [MIT license](https://choosealicense.com/licenses/mit/).
