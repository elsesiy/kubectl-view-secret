#!/usr/bin/env bash

set -euo pipefail

CLUSTER_NAME="kvs-test"

# Check if kind & kubectl are installed
which kind &>/dev/null || { echo "failed to find 'kind' binary, please install it" && exit 1; }
which kubectl &>/dev/null || { echo "failed to find 'kubectl' binary, please install it" && exit 1; }

# Ensure the cluster exists
[[ $(kind get clusters) == *$CLUSTER_NAME* ]] || kind create cluster --name $CLUSTER_NAME

# Set context in case there are mulitple
kubectl config set-context kind-${CLUSTER_NAME}

# Seed test secrets

## secret 'test' in namespace 'default' (multiple keys)
kubectl apply -f - <<EOF
apiVersion: v1
data:
  key1: dmFsdWUxCg==
  key2: dmFsdWUyCg==
kind: Secret
metadata:
  name: test
  namespace: default
type: Opaque
EOF

## secret 'test2' in namespace 'default' (single key)
kubectl apply -f - <<EOF
apiVersion: v1
data:
  key1: dmFsdWUx
kind: Secret
metadata:
  name: test2
  namespace: default
type: Opaque
EOF

## 'another' namespace
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: another
EOF

## secret 'gopher' in namespace 'another' (single key)
kubectl apply -f - <<EOF
apiVersion: v1
data:
  foo: YmFy
kind: Secret
metadata:
  name: gopher
  namespace: another
type: Opaque
EOF

## 'empty' namespace
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: empty
EOF
