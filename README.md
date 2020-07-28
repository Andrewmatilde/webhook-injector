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
`cd envoy`
run `bash build.sh` to build envoy image

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
myapp-8484956db4-xvknp                                 2/2     Running       0          20s
```
Then
`kubectl port-forward --address 0.0.0.0 pod/myapp-8484956db4-xvknp 8877:15006`
Open 127.0.0.1:8877 in broswer refresh the web page, and there will be a 1 second delay.
## Sidecar

`Sidecar` is defined by `config` in ./pkg/config/example/config.yaml`.
