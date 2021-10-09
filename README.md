# kidle

[![snapshot](https://github.com/kidle-dev/kidle/actions/workflows/snapshot.yaml/badge.svg)](https://github.com/kidle-dev/kidle/actions/workflows/snapshot.yaml)

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

## Docker

The [operator](cmd/operator) builds are published to [docker](https://hub.docker.com/r/kidledev/kidle-operator).

Following tags are maintained:

  * `vx.y.z`: Images that are build from the tagged versions within Github. Always unique.
  * `vx.y`: Represents latest `.z` revision.
  * `vx`: Represents latest `.y.z` revision.

```
docker pull kidledev/kidle-operator:<tag>
```
