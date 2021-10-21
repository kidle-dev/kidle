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

For now you can deploy kidle using the `deploy` target:

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

## Documentation

Here is a [user guide](docs/userguide.md).

## Contact

You can find me on the [kidle-dev slack](https://kidle-dev.slack.com/archives/C02JXP2JTK2)
