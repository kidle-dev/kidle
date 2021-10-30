# Kidle Monitoring

## Prometheus installation


Based on [kube-prometheus](https://github.com/prometheus-operator/kube-prometheus)
```
git clone git@github.com:prometheus-operator/kube-prometheus.git
```


edit file `example.jsonnet`

```jsonnet
    values+:: {
      common+: {
        namespace: 'monitoring',
      },
      prometheus+: {
        namespaces+: ['kidle-system'],
      }
    },
```

generate manifests:
```
docker run --rm -v $(pwd):$(pwd) --workdir $(pwd) quay.io/coreos/jsonnet-ci ./build.sh example.jsonnet
```

apply manifests:
```
# Create the namespace and CRDs, and then wait for them to be available before creating the remaining resources
kubectl create -f manifests/setup
until kubectl get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done
kubectl create -f manifests/
```

## Access the dashboards

Prometheus:
```
kubectl --namespace monitoring port-forward svc/prometheus-k8s 9090
```
Then access via http://localhost:9090

Grafana:
```
kubectl --namespace monitoring port-forward svc/grafana 3000
```
Then access via http://localhost:3000 and use the default grafana user:password of admin:admin.

Alert Manager:
```
kubectl --namespace monitoring port-forward svc/alertmanager-main 9093
```
Then access via http://localhost:9093

