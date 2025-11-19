# Troubleshooting Guide

## Common Issues

### 1. Webhook Not Intercepting Pods

**Symptoms:**
- Pods are created without labels
- No logs in webhook pod

**Diagnosis:**
```bash
# Check webhook configuration
kubectl get mutatingwebhookconfiguration mutating-webhook -o yaml

# Check if webhook pod is running
kubectl get pods -n webhook-system

# Check namespace selector
kubectl describe mutatingwebhookconfiguration mutating-webhook
```

**Solutions:**
- Verify namespace selector if configured
- Check webhook service is accessible
- Ensure certificate is valid

### 2. Certificate Errors

**Symptoms:**
- Error: `x509: certificate signed by unknown authority`
- Pods fail to create

**Diagnosis:**
```bash
# Check certificate job
kubectl get job -n webhook-system
kubectl logs job/mutating-webhook -n webhook-system

# Check certificate secret
kubectl get secret webhook-server-cert -n webhook-system
kubectl describe secret webhook-server-cert -n webhook-system
```

**Solutions:**
```bash
# Regenerate certificates
helm upgrade mutating-webhook ./helm-chart \
  --set certificate.generate=true \
  --force

# Or delete and reinstall
helm uninstall mutating-webhook -n webhook-system
helm install mutating-webhook ./helm-chart -n webhook-system
```

### 3. Webhook Timeout

**Symptoms:**
- Pods take long to create
- Error: `context deadline exceeded`

**Diagnosis:**
```bash
# Check webhook logs
kubectl logs -l app.kubernetes.io/name=mutating-webhook -n webhook-system

# Check webhook latency
kubectl get mutatingwebhookconfiguration mutating-webhook -o yaml | grep timeoutSeconds
```

**Solutions:**
```yaml
# Increase timeout in values.yaml
webhook:
  timeoutSeconds: 10

# Or update directly
kubectl patch mutatingwebhookconfiguration mutating-webhook \
  --type='json' -p='[{"op": "replace", "path": "/webhooks/0/timeoutSeconds", "value": 10}]'
```

### 4. Labels Not Being Added

**Symptoms:**
- Webhook runs but labels don't appear

**Diagnosis:**
```bash
# Check webhook logs for patch operations
kubectl logs -l app.kubernetes.io/name=mutating-webhook -n webhook-system | grep "Applied patch"

# Check if labels already exist on pod
kubectl get pod <pod-name> -o jsonpath='{.metadata.labels}'

# Check ConfigMap
kubectl get configmap mutating-webhook-config -n webhook-system -o yaml
```

**Solutions:**
- Verify labels in ConfigMap match expected values
- Check if pod already has the labels (webhook skips existing labels)
- Restart webhook pod: `kubectl rollout restart deployment mutating-webhook -n webhook-system`

### 5. Permission Errors

**Symptoms:**
- Error: `forbidden: User "system:serviceaccount:webhook-system:mutating-webhook" cannot...`

**Diagnosis:**
```bash
# Check service account
kubectl get serviceaccount -n webhook-system

# Check RBAC
kubectl describe clusterrole mutating-webhook
```

**Solutions:**
- Reinstall with proper RBAC: `helm upgrade mutating-webhook ./helm-chart`

## Debug Mode

Enable verbose logging:

```yaml
# values.yaml
webhook:
  extraEnv:
    - name: LOG_LEVEL
      value: "debug"
```

## Health Checks

```bash
# Check webhook health endpoint
kubectl port-forward -n webhook-system svc/mutating-webhook 8443:443
curl -k https://localhost:8443/health

# Expected response: {"status":"healthy"}
```

## Testing Without Production Impact

Test in a separate namespace:

```bash
# Create test namespace
kubectl create namespace webhook-test

# Label it for webhook
kubectl label namespace webhook-test webhook-enabled=true

# Create test pod
kubectl run test-pod --image=nginx -n webhook-test

# Check labels
kubectl get pod test-pod -n webhook-test --show-labels

# Clean up
kubectl delete namespace webhook-test
```

## Collecting Debug Information

```bash
# Webhook logs
kubectl logs -l app.kubernetes.io/name=mutating-webhook -n webhook-system --tail=100 > webhook-logs.txt

# Webhook configuration
kubectl get mutatingwebhookconfiguration mutating-webhook -o yaml > webhook-config.yaml

# Deployment info
kubectl describe deployment mutating-webhook -n webhook-system > deployment-info.txt

# Events
kubectl get events -n webhook-system --sort-by='.lastTimestamp' > events.txt
```

## Getting Help

If you're still having issues:

1. Check existing GitHub issues
2. Create a new issue with:
   - Debug information from above
   - Steps to reproduce
   - Expected vs actual behavior
   - Kubernetes version
   - Helm version
