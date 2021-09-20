# Kidle user guide

Kidle is a Kubernetes operator that provides an idling feature to automatically idle or wakeup workloads.

## Introduction

The Kidle CRD introduces a new kubernetes resource named `IdlingResource`. A single `IdlingResource` instance will drive the idle status of a single workload. 

The supported workload types are:

- Deployment
- StatefulSet
- CronJob

The `IdlingResource` has a boolean field `spec.idle`. 
When the value is `true`, the Kidle operator will idle the workload. 
Set it back to `false` to wakeup the workload. 

During the **idling phase**, Kidle's operator will scale down the deployments and the statefulsets to 0.
The cronjobs `spec.suspend` field is set to `true`. 
Before the scale down, Kidle saves the previous replicas state for the wakeup phase.

The **wakeup phase** restores the previous replicas state on deployments and statefulsets. 
The cronjobs `spec.suspend` field is set back to `false`.

When deleting an `IdlingResource` objet on a idled workload, the operator will wakeup the referenced workload.

There are several idling strategies:

1. manual: play around with the idle field
1. cronjob: idle and wakeup times defined by cron expressions

## Installation
### Prerequisites
- golang >=1.16
- [k3d](https://github.com/rancher/k3d)

### Run the operator on a local k3s cluster
```bash
# start a local k3s cluster
make k3s-registry k3s-create k3s-kubeconfig

# set proper KUBECONFIG env var
export KUBECONFIG=$(pwd)/kube.config

# Install the CRD
make install

# Start the operator
make WHAT=operator run
```

### Kidlectl

```bash
# Run kidlectl using golang
make WHAT=kidlectl run

# or build kidlectl
make WHAT=kidlectl build
./cmd/kidlectl/bin/kidlectl
```

The output is:
```
Please specify one command of: idle or wakeup
Usage:
  kidlectl [OPTIONS] <idle | wakeup>

Application Options:
      --kubeconfig= path to Kubernetes config file [$KUBECONFIG]

Help Options:
  -h, --help        Show this help message

Available commands:
  idle    idle the referenced object of an IdlingResource (aliases: i)
  wakeup  wakeup the referenced object of an IdlingResource (aliases: w)
```

## Quickstart

First, create a Deployment:
```bash
$ kubectl create deploy --replicas=2 --image=stefanprodan/podinfo podinfo
```

Then, create an `IdlingResource` object. Here is the most basic form which enables **manual idling of a Deployment**:

```yaml
apiVersion: kidle.kidle.dev/v1beta1
kind: IdlingResource
metadata:
  name: podinfo
spec:
  # Reference of the workload to idle
  idlingResourceRef:
    apiVersion: apps/v1
    kind: Deployment
    name: podinfo

  # Idle or not?
  idle: false
```

- This create an `IdlingResource` object that references the Deployment named podinfo.
- The initial idle status is `false`.

At this point, the Deployment still have 2 replicas:

```bash
$ kubectl get pods                                                                              
NAME                       READY   STATUS    RESTARTS   AGE
podinfo-7fbb45ccfc-kg42f   1/1     Running   0          6s
podinfo-7fbb45ccfc-mlrzf   1/1     Running   0          6s
```

You can **get the idling status** on the `IdlingResource` object:
```bash
$ kubectl get idlingresources
NAME      IDLE    REFKIND      REFNAME
podinfo   false   Deployment   podinfo
```

To **idle the Deployment**, you can either:

1. edit the `IdlingResource` and set the `spec.idle` field to `false`
  ```bash
  $ kubectl edit ir/podinfo
  ```
2. or use `kidlectl` for a simple single line command:
  ```bash
  $ kidlectl idle podinfo
  2021-09-20T12:52:12.996+0200	INFO	idling	{"namespace": "kidle-demo", "name": "podinfo"}
  2021-09-20T12:52:13.002+0200	INFO	scaled to 0	{"namespace": "kidle-demo", "name": "podinfo"}  
  ```

Now **the Deployment is idle** and have no pods anymore:
```bash
$ kubectl get pods
No resources found in kidle-demo namespace.

$ kubectl get deploy
NAME      READY   UP-TO-DATE   AVAILABLE   AGE
podinfo   0/0     0            0           8m53s

$ kubectl get idlingresources
NAME      IDLE    REFKIND      REFNAME
podinfo   true   Deployment   podinfo
```

You can track the Kidle operator activity on the `IdlingResource` events:
```bash
$ kubectl describe ir/podinfo
...
Events:
  Type    Reason             Age   From                       Message
  ----    ------             ----  ----                       -------
  Normal  Added              50s   idlingresource-controller  Object finalizer is added
  Normal  ScalingDeployment  19s   idlingresource-controller  Scaled to 0
  Normal  ScalingDeployment  2s    idlingresource-controller  Scaled to 2

```

## Cronjob idle strategy

The cronjob idle strategy schedules idle and wakeup phases using a cron expression:

```yaml
apiVersion: kidle.kidle.dev/v1beta1
kind: IdlingResource
metadata:
  name: podinfo
spec:
  # Reference of the workload to idle
  idlingResourceRef:
    apiVersion: apps/v1
    kind: Deployment
    name: podinfo

  # Idle or not?
  idle: false

  # Scheduled idle time using cronjob
  idlingStrategy:
    cronStrategy:
      schedule: "*/2 * * * *"

  # Scheduled wakeup time using cronjob
  wakeupStrategy:
    cronStrategy:
      schedule: "1-59/2 * * * *"
```

The Kidle operator will create kubernetes cronjobs:
```bash
$ kubectl get cronjobs
NAME                   SCHEDULE         SUSPEND   ACTIVE   LAST SCHEDULE   AGE
kidle-podinfo-idle     */2 * * * *      False     0        98s             6m43s
kidle-podinfo-wakeup   1-59/2 * * * *   False     0        38s             6m43s
```

The cronjob is based on the `kidlectl` image and run a basic `kidlectl <idle|wakeup> podinfo` command inside the job pod:

```bash
$ kubectl logs jobs/kidle-podinfo-wakeup-27202409                                                                                            
2021-09-20T13:29:02.241Z	INFO	waking up	{"namespace": "kidle-demo", "name": "podinfo"}
2021-09-20T13:29:02.265Z	INFO	waked up	{"namespace": "kidle-demo", "name": "podinfo"}
```

In order to make `kidlectl` work inside a pod, a dedicated service account, role and role binding are created per `IdlingResource`:

```bash
$ kubectl get serviceaccount,role,rolebinding                                                                                                            
NAME                              SECRETS   AGE
serviceaccount/default            1         6h13m
serviceaccount/kidle-podinfo-sa   1         64m

NAME                                                CREATED AT
role.rbac.authorization.k8s.io/kidle-podinfo-role   2021-09-20T12:38:22Z

NAME                                                     ROLE                      AGE
rolebinding.rbac.authorization.k8s.io/kidle-podinfo-rb   Role/kidle-podinfo-role   64m
```


## Supported workloads
Here are examples for each workload supported by Kidle:

**Deployment**:
```yaml
spec:
  idlingResourceRef:
    apiVersion: apps/v1
    kind: Deployment
    name: deployment-name
```

**StatefulSet**:
```yaml
spec:
  idlingResourceRef:
    apiVersion: apps/v1
    kind: StatefulSet
    name: statefulset-name
```

**CronJob**:
```yaml
spec:
  idlingResourceRef:
    apiVersion: batch/v1
    kind: CronJob
    name: cronjob-name
```

or:
```yaml
spec:
  idlingResourceRef:
    apiVersion: batch/v1beta1
    kind: CronJob
    name: cronjob-name
```

