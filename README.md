# payschedule-operator

A Kubernetes operator, written in Go with [kubebuilder](https://book.kubebuilder.io/) and controller-runtime,
that runs scheduled payment jobs. You describe a job with a `PaySchedule` custom resource,
and the controller creates and keeps a matching `CronJob` in sync.

## How it works

```
PaySchedule (pay.mrahman.xyz/v1)  ──owns──▶  CronJob (batch/v1)  ──▶  Job  ──▶  Pod
        ▲                                         │
        └──────── status.lastScheduled ◀──────────┘
```

On every reconcile, the controller:

1. Fetches the `PaySchedule`. If it's gone, there's nothing to do.
2. Creates or updates the `CronJob` named in `spec.cronName`, using `controllerutil.CreateOrUpdate`.
   It sets only the fields it owns (schedule, image, command, restart policy), so defaults filled in by the API server are kept.
3. Sets the `PaySchedule` as the controller owner of the `CronJob`. Deleting the `PaySchedule` garbage-collects the `CronJob`.
4. Copies `CronJob.status.lastScheduleTime` into `PaySchedule.status.lastScheduled`.

The controller watches `PaySchedule` objects and their owned `CronJob`s (`Owns`). So:

- a spec change rolls out to the `CronJob`
- a deleted or hand-edited `CronJob` is recreated or reverted
- the status follows every new schedule run

RBAC is generated from `+kubebuilder:rbac` markers and limited to the resources the controller touches.

## The PaySchedule API

```yaml
apiVersion: pay.mrahman.xyz/v1
kind: PaySchedule
metadata:
  name: payschedule-sample
spec:
  cronName: pay-cron            # name of the CronJob to manage
  cronSchedule: "*/1 * * * *"   # standard cron syntax
  jobImage: busybox:latest      # container image of the job
  jobCmd: echo Hello from PaySchedule!  # run via /bin/sh -c
```

All spec fields are required and must be non-empty, which the CRD schema enforces.

```sh
$ kubectl get payschedules
NAME                 CRONJOB    SCHEDULE      AGE   LAST SCHEDULED
payschedule-sample   pay-cron   */1 * * * *   3m    20s
```

## Getting Started

### Prerequisites
- Go 1.20+
- Docker
- kubectl
- A Kubernetes cluster, e.g. [kind](https://kind.sigs.k8s.io/)

### Run locally against a cluster

```sh
kind create cluster
make install          # install the CRD
make run              # run the controller from your machine
kubectl apply -k config/samples/
kubectl get payschedules,cronjobs
```

### Deploy to the cluster

```sh
make docker-build IMG=payschedule-operator:dev
kind load docker-image payschedule-operator:dev
make deploy IMG=payschedule-operator:dev
```

### Clean up

```sh
kubectl delete -k config/samples/
make undeploy
make uninstall
```

## Testing

```sh
make test
```

The tests use [envtest](https://book.kubebuilder.io/reference/envtest.html), a real kube-apiserver and etcd with no nodes.
They check that:

- the `CronJob` is created with the right spec and owner reference
- spec changes reach the `CronJob`
- a deleted `CronJob` is recreated
- a missing `PaySchedule` is not treated as an error
- invalid specs are rejected by the CRD schema

## Limitations

- Changing `spec.cronName` creates a new `CronJob`. The old one is only removed when the `PaySchedule` is deleted.
- A bad cron expression is rejected when the `CronJob` is created, and shows up only in the controller logs.
  It isn't reported in the `PaySchedule` status (no status conditions yet).

## License

Copyright 2023 Masudur Rahman.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

