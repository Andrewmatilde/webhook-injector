go mod download
go mod vendor
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sidecar-inject-server .
docker build --no-cache -t yisaer/sidecar-inject-server:v1 .
rm -rf sidecar-inject-server

kind load docker-image yisaer/sidecar-inject-server:v1

kubectl delete deployment myapp

# helm install sidecar-init install/helm/sidecar-inject-init --namespace sidecar-system 

# helm install sidecar-server install/helm/sidecar-inject-server --namespace sidecar-system --values install/helm/sidecar-inject-server/values.yaml 