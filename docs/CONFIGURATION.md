# Configuration Guide

## Label Configuration Methods

### Method 1: ConfigMap (Recommended)

The default and recommended method for configuring labels.

**values.yaml:**
```yaml
webhook:
  useConfigFile: true
  labels:
    mutated: "true"
    team: "platform"
    environment: "production"
```

**Advantages:**
- Easy to update without pod restart
- Supports many labels
- Clean separation of config

**Update labels:**
```bash
kubectl edit configmap mutating-webhook-config -n webhook-system
kubectl rollout restart deployment mutating-webhook -n webhook-system
```

### Method 2: JSON Environment Variable

Pass all labels as a single JSON environment variable.

**values.yaml:**
```yaml
webhook:
  useConfigFile: false
  labels:
    mutated: "true"
    team: "backend"
```

This creates: `WEBHOOK_LABELS='{"mutated":"true","team":"backend"}'`

### Method 3: Individual Environment Variables

Pass each label as a separate environment variable with `LABEL_` prefix.

**values.yaml:**
```yaml
webhook:
  useConfigFile: false
  extraEnv:
    - name: LABEL_mutated
      value: "true"
    - name: LABEL_team
      value: "platform"
```

## Namespace Filtering

### Apply to All Namespaces (Default)

```yaml
webhook:
  namespaceSelector: {}
```

### Apply to Labeled Namespaces Only

```yaml
webhook:
  namespaceSelector:
    matchLabels:
      webhook-enabled: "true"
```

Then label namespaces:
```bash
kubectl label namespace production webhook-enabled=true
kubectl label namespace staging webhook-enabled=true
```

### Exclude Specific Namespaces

```yaml
webhook:
  namespaceSelector:
    matchExpressions:
    - key: webhook-enabled
      operator: NotIn
      values:
      - "false"
```

## Failure Policy

### Ignore (Default - Recommended for Testing)

```yaml
webhook:
  failurePolicy: Ignore
```

Pods are created even if webhook fails.

### Fail (Production)

```yaml
webhook:
  failurePolicy: Fail
```

Pod creation is blocked if webhook fails.

## Resource Configuration

### Development

```yaml
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi
```

### Production

```yaml
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 200m
    memory: 256Mi
```

## High Availability

```yaml
replicaCount: 3

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
    - labelSelector:
        matchLabels:
          app.kubernetes.io/name: mutating-webhook
      topologyKey: kubernetes.io/hostname
```

## Custom TLS Certificates

### Disable Auto-Generation

```yaml
certificate:
  generate: false
  secretName: my-custom-cert
```

### Create Custom Certificate

```bash
# Generate certificates
openssl req -x509 -newkey rsa:4096 -keyout tls.key -out tls.crt -days 365 -nodes

# Create secret
kubectl create secret tls my-custom-cert \
  --cert=tls.crt \
  --key=tls.key \
  -n webhook-system
```

## Advanced Label Configuration

### Conditional Labels Based on Namespace

Modify `main.go`:

```go
func createPatch(pod *corev1.Pod) []patchOperation {
    labelsToAdd := config.Labels
    
    // Add namespace-specific labels
    if pod.Namespace == "production" {
        labelsToAdd["environment"] = "prod"
        labelsToAdd["sla"] = "high"
    } else if pod.Namespace == "staging" {
        labelsToAdd["environment"] = "staging"
        labelsToAdd["sla"] = "medium"
    }
    
    // ... rest of the code
}
```

### Dynamic Labels from Pod Annotations

```go
// Read labels from pod annotations
if val, ok := pod.Annotations["webhook.example.com/extra-labels"]; ok {
    var extraLabels map[string]string
    json.Unmarshal([]byte(val), &extraLabels)
    for k, v := range extraLabels {
        labelsToAdd[k] = v
    }
}
```

Then use:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  annotations:
    webhook.example.com/extra-labels: '{"custom":"value"}'
```
