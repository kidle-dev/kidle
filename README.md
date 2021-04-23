# kidle

Kidle is a kubernetes idling feature to automatically idle or wakeup workloads.

Main features:

[x] idle and wakeup Deployments, StatefulSets and CronJobs
[-] idle and wakeup at specified time
[-] shutdown after some idle time
[-] automatic wakeup on call
[-] fancy UI



## Deployment

### Deploying with Helm


## Development

## Running the operator locally

```shell
make k3s-registry k3s-create k3s-kubeconfig
make install
make run
```

## Building/Pushing the operator image

```shell
export repo=kidle #replace with yours
make docker-build IMG=$repo/kidle-operator:latest
make docker-push IMG=$repo/kidle-operator:latest
```
