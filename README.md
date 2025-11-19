# Kubernetes Mutating Webhook

A production-ready Kubernetes mutating admission webhook that dynamically adds labels to pods.

![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Helm](https://img.shields.io/badge/helm-%230F1689.svg?style=for-the-badge&logo=helm&logoColor=white)

## Features

- ✅ **Dynamic Label Configuration** - Configure labels via ConfigMap, environment variables, or JSON
- ✅ **Zero Downtime Updates** - Change labels without rebuilding the image
- ✅ **Production Ready** - Includes health checks, RBAC, and security best practices
- ✅ **Helm Chart** - Easy deployment with configurable values
- ✅ **Automatic TLS** - Certificate generation via Kubernetes jobs


## Quick Start

### Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- kubectl configured
- Docker (for building images)

### Installation

#### Option 1: Minikube (Local Development)

```bash
# Start Minikube
minikube start

# Build the image in Minikube
make docker-build-minikube

# Install the webhook
make helm-install

# Test it
make test-webhook
```

#### Option 2: Standard Kubernetes Cluster

```bash
# Build and push image to your registry
export REGISTRY=docker.io/yourusername
make docker-build
make docker-push

# Update values.yaml with your image
# Then install
helm install mutating-webhook ./helm-chart -n webhook-system --create-namespace
```

### Verify Installation

```bash
# Check webhook pod
kubectl get pods -n webhook-system

# Check webhook configuration
kubectl get mutatingwebhookconfiguration

# View logs
kubectl logs -l app.kubernetes.io/name=mutating-webhook -n webhook-system -f
```

### Test the Webhook

```bash
# Create a test pod
kubectl run test-pod --image=nginx

# Check if labels were added
kubectl get pod test-pod --show-labels

# You should see: mutated=true
```

## Configuration

### Configure Labels

Edit `helm-chart/values.yaml`:

```yaml
webhook:
  useConfigFile: true
  labels:
    mutated: "true"
    team: "platform"
    environment: "production"
    owner: "devops"
  topologySpreadConstraints:
  - labelSelector:
      matchLabels:
        app: workload-split
    maxSkew: 1
    topologyKey: capacity-spread
    whenUnsatisfiable: DoNotSchedule    
```

### Dynamic Label Updates


**Method: Edit ConfigMap**
```bash
kubectl edit configmap mutating-webhook-config -n webhook-system
kubectl rollout restart deployment mutating-webhook -n webhook-system
```


## Architecture

```
┌─────────────────┐
│   API Server    │
└────────┬────────┘
         │
         │ AdmissionReview
         ▼
┌─────────────────┐
│ Mutating        │
│ Webhook         │◄───── ConfigMap (labels.json, topologySpreadConstraints.json)
│ Server          │
└────────┬────────┘
         │
         │ Patch Response
         ▼
┌─────────────────┐
│   Pod with      │
│   Added Labels  │
│   Added topology│
└─────────────────┘
```

## Development

### Build Locally

```bash

# Build Docker image
make docker-build

make docker-build-minikube #for mac using minikube locally
```

### Install / Uninstall Helm
```bash

make helm-install #for install and upgrade

make helm-uninstall #to uninstall

```


### Test
```bash

# test pod without topologycontrainst in the podspecs
make test-webhook-without-topology

# test pod with topologycontrainst in the podspecs
make test-webhook-with-topology

# test pod with similar topologycontrainst in the podspecs as in the values yaml webhook.topologySpreadConstraints
make test-webhook-with-dupe-topology


```
**Note**: the test pod yaml can be found under ./examples/ folder

### Project Structure

```
k8s-mutating-webhook/
├── main.go                 # Webhook server code
├── go.mod                  # Go dependencies
├── Dockerfile             # Container image definition
├── Makefile              # Build automation
└── helm-chart/           # Helm chart
    ├── Chart.yaml
    ├── values.yaml
    └── templates/
        ├── deployment.yaml
        ├── service.yaml
        ├── mutatingwebhookconfiguration.yaml
        └── ...
```

## Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `webhook.labels` | Labels to add to pods | `{mutated: "true"}` |
| `webhook.useConfigFile` | Use ConfigMap for labels | `true` |
| `webhook.topologySpreadConstraints` | topologySpreadConstraints to be added to the pod spec | `{...}` |

See [helm-chart/values.yaml](helm-chart/values.yaml) for all options.

## Troubleshooting

### Webhook Not Working

```bash
# Check webhook logs
kubectl logs -l app.kubernetes.io/name=mutating-webhook -n webhook-system --tail=100

# Check webhook configuration
kubectl describe mutatingwebhookconfiguration mutating-webhook

# Check certificate
kubectl get secret webhook-server-cert -n webhook-system
```

### Certificate Issues

```bash
# Check certificate generation job
kubectl get job -n webhook-system
kubectl logs job/mutating-webhook -n webhook-system
```


