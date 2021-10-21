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

You can deploy kidle using the `deploy` target. 
It expects a cluster-admin role and creates a `kidle-system` namespace.
You can select the release you want by setting the `TAG=` as an environment variable.

```
$ TAG=main make deploy
$ kubectl get deploy -n kidle-system
NAME                       READY   UP-TO-DATE   AVAILABLE   AGE
kidle-controller-manager   1/1     1            1           13d
```

To uninstall:
```
make undeploy
```

If you want to see the kubernetes manifests before applying:
```
make deploy-view
```

## Documentation

Here is a [user guide](docs/userguide.md).

## Contact

You can find me on the [kidle-dev slack](https://kidle-dev.slack.com/archives/C02JXP2JTK2)
