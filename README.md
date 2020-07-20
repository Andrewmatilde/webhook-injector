# README

`webhook-injector` is a simple server to inject containers.

It's easy to use and also offer container selector ability.

## Prerequisites

Kubernetes 1.9.0 or above with the `admissionregistration.k8s.io/v1beta1` API enabled. Verify that by the following command:
```
kubectl api-versions | grep admissionregistration.k8s.io/v1beta1
```
The result should be:
```
admissionregistration.k8s.io/v1beta1
```

In addition, the `MutatingAdmissionWebhook` admission controllers should be added and listed in the correct order in the admission-control flag of kube-apiserver.

## Build Image

`webhook-injector` is managed by `go mod`

run `build.sh` to build docker image

## How to Install

you can install sidecar-inject by `Helm`

First install `sidecar-inejct-init` 
```yaml
helm install install/helm/sidecar-inject-init --name sidecar-init --namespace sidecar-system 
```

Then install `sidecar-inject-server`
```yaml
helm install install/helm/sidecar-inject-server --name sidecar-server --namespace sidecar-system --values install/helm/sidecar-inject-server/values.yaml 
```

## How to test

Label the `namespace` and create a `deployment` to test sidecar-inject-server

```
kubectl label namespace default sidecar-injector=enabled
kubectl apply -f example/
```
And Here is the result
```
$ kubectl get pod
NAME                                                    READY   STATUS        RESTARTS   AGE
sleep-578649fc85-xcg2r                                  2/2     Running       0          20s
```

## Sidecar

`Sidecar` is defined by `config` in ./pkg/config/example/config.yaml`.
