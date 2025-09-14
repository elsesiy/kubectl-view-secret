#!/usr/bin/env bash

set -euo pipefail

CLUSTER_NAME="kvs-test"

# Check if kind & kubectl are installed
which kind &>/dev/null || { printf '%s\n' "failed to find 'kind' binary, please install it" && exit 1; }
which kubectl &>/dev/null || { printf '%s\n' "failed to find 'kubectl' binary, please install it" && exit 1; }

# Ensure the cluster exists
[[ "$(kind get clusters)" == *$CLUSTER_NAME* ]] || kind create cluster --name "$CLUSTER_NAME"

# Set context in case there are mulitple
kubectl config set-context "kind-${CLUSTER_NAME}"

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

## 'helm' namespace
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: helm
EOF

## helm secret 'test3' in namespace 'helm' (single key)
sec=$(printf '%s' "helm-test" | gzip -c | base64 | base64)
kubectl apply -f - <<EOF
apiVersion: v1
data:
  release: "$sec"
kind: Secret
metadata:
  name: test3
  namespace: helm
type: helm.sh/release.v1
EOF

## TLS secret 'tls-secret' in namespace 'default'
# Generate dummy TLS cert and key
openssl req -x509 -newkey rsa:2048 -keyout /tmp/tls_key.pem -out /tmp/tls_cert.pem -days 1 -nodes -subj "/CN=example.com" 2>/dev/null
tls_cert_b64=$(base64 -w 0 < /tmp/tls_cert.pem)
tls_key_b64=$(base64 -w 0 < /tmp/tls_key.pem)
rm /tmp/tls_cert.pem /tmp/tls_key.pem
kubectl apply -f - <<EOF
apiVersion: v1
data:
  tls.crt: "$tls_cert_b64"
  tls.key: "$tls_key_b64"
kind: Secret
metadata:
  name: tls-secret
  namespace: default
type: kubernetes.io/tls
EOF

## Docker config secret 'docker-secret' in namespace 'default'
kubectl apply -f - <<EOF
apiVersion: v1
data:
  .dockerconfigjson: "$(printf '%s' '{"auths":{"registry.example.com":{"username":"user","password":"pass","auth":"dXNlcjpwYXNz"}}}' | base64 -w 0)"
kind: Secret
metadata:
  name: docker-secret
  namespace: default
type: kubernetes.io/dockerconfigjson
EOF

## Legacy Docker config secret 'docker-legacy-secret' in namespace 'default'
kubectl apply -f - <<EOF
apiVersion: v1
data:
  .dockercfg: "$(printf '%s' '{"registry.example.com":{"username":"legacy-user","password":"legacy-pass","auth":"bGVnYWN5LXVzZXI6bGVnYWN5LXBhc3M="}}' | base64 -w 0)"
kind: Secret
metadata:
  name: docker-legacy-secret
  namespace: default
type: kubernetes.io/dockercfg
EOF

## SSH auth secret 'ssh-secret' in namespace 'default'
kubectl apply -f - <<EOF
apiVersion: v1
data:
  ssh-privatekey: "$(printf '%s' "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzasupersecretprivatekeyzc2gtZW\n-----END OPENSSH PRIVATE KEY-----" | base64 -w 0)"
kind: Secret
metadata:
  name: ssh-secret
  namespace: default
type: kubernetes.io/ssh-auth
EOF

## Basic auth secret 'basic-auth-secret' in namespace 'default'
kubectl apply -f - <<EOF
apiVersion: v1
data:
  username: "$(printf '%s' "admin" | base64 -w 0)"
  password: "$(echo "secret123" | base64 -w 0)"
kind: Secret
metadata:
  name: basic-auth-secret
  namespace: default
type: kubernetes.io/basic-auth
EOF

# Create ClusterRole for secret access
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: secret-reader
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list"]
EOF

# Bind ClusterRole to user 'gopher'
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: gopher-secret-reader
subjects:
- kind: User
  name: gopher
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: secret-reader
  apiGroup: rbac.authorization.k8s.io
EOF

# Bind ClusterRole to group 'golovers'
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: golovers-secret-reader
subjects:
- kind: Group
  name: golovers
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: secret-reader
  apiGroup: rbac.authorization.k8s.io
EOF

