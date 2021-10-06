# kidle

[![build](https://github.com/kidle-dev/kidle/actions/workflows/dev-branch.yaml/badge.svg?branch=main)](https://github.com/kidle-dev/kidle/actions/workflows/snapshot.yaml)

Kidle is a kubernetes idling feature to automatically idle or wakeup workloads.

Main features:

- [x] idle and wakeup Deployments, StatefulSets and CronJobs
- [x] idle and wakeup at specified time
- [ ] shutdown after some idle time
- [ ] automatic wakeup on call
- [ ] fancy UI

## Demo

[![asciicast](https://asciinema.org/a/ucJjxq0BmygzZdjTozNgbbf6o.svg)](https://asciinema.org/a/ucJjxq0BmygzZdjTozNgbbf6o)

Demo commands:
```bash
# let's create a deployment
kubectl create deploy --image=stefanprodan/podinfo podinfo

# create an IdlingResource for that deployment
bat 01-manual-idlingresource.yaml
kubectl apply -f 01-manual-idlingresource.yaml

# display kidle status
kubectl get idlingresources

# idle manually the deployment then watch the result
kubectl edit ir/podinfo
kubectl get ir,deploy

# use kidlectl to idle or wakeup more easily
kidlectl wakeup podinfo
kubectl get ir,deploy
 
```

## Deployment

For now, kidle is in early stages. When ready, Kidle will be deployed using a helm chart or kustomize.

## Documentation

Here is a [user guide](docs/userguide.md).
