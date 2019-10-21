# kubectl-view-secret
[![Go Report Card](https://goreportcard.com/badge/github.com/elsesiy/kubectl-view-secret)](https://goreportcard.com/report/github.com/elsesiy/kubectl-view-secret)
[![Build Status](https://travis-ci.org/elsesiy/kubectl-view-secret.svg?branch=master)](https://travis-ci.org/elsesiy/kubectl-view-secret)
[![Twitter](https://img.shields.io/badge/twitter-@elsesiy-blue.svg)](http://twitter.com/elsesiy)
[![GitHub release](https://github.com/elsesiy/kubectl-view-secret/releases)](https://img.shields.io/github/v/release/elsesiy/kubectl-view-secret.svg)

This plugin allows for easy secret decoding. Useful if you want to see what's inside of a secret without always go throug the following:
1. `kubectl get secret <secret> -o yaml`
2. Copy base64 encoded secret
3. `echo "b64string" | base64 -d`

Instead you can now do:

    kubectl view-secret <secret>
    kubectl view-secret <secret> -n/--namespace <ns> # override namespace

## Build

    # Clone this repository (or your fork)
    git clone https://github.com/elsesiy/kubectl-view-secret
    cd kubectl-view-secret
    make

## License

This repository is available under the [MIT license](https://choosealicense.com/licenses/mit/).