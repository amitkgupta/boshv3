# BOSH v3 API (theoretical)

[![license](https://img.shields.io/github/license/amitkgupta/boshv3.svg)](LICENSE)

The goal of this project is to explore:

- a concept: a "v3" API for [BOSH](https://bosh.io/) comprised of consistent, modular resources.
- a technology: [Kubebuilder](https://book.kubebuilder.io/) and Kubernetes [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) (CRDs)

This repository contains source code and [make](https://www.gnu.org/software/make/manual/make.html) tasks
to build and install CRDs and controllers to a Kubernetes cluster. They extend the Kubernetes API to expose
resources to manage tenancy to multiple BOSH Directors and enable tenants to create BOSH resources. In some
sense this API extension is a shim in front of the BOSH API to explore what a different style of API for
BOSH could be like.

## Table of Contents

- [Install](#install)
- [Concepts](#concepts)
- [Usage](#usage)
- [Specification](#specification)
- [TODO](#todo)
- [Issues](#issues)
- [Development](#development)
- [License](#license)

## Install

You will need to have some prerequisites installed such as `go`, `make`, `kustomize`, and `kubectl`. To
install this to a Kubernetes cluster, make sure `$GOPATH` is properly set, make sure `kubectl` is properly
targetting the cluster, and run:

```
$ go get githbu.com/amitkgupta/boshv3/...

$ cd $GOPATH/src/github.com/amitkgupta/boshv3

$ make install # 🤞🏾
```

For this installation to actually be useable, you will need access to one or more BOSH Directors with
admin-level access to its UAA. It will also be necessary for those BOSH Directors to be addressable
and routable from the controller pods deployed to the Kubernetes cluster.

### Local Installation

You can install this locally using [minikube](https://github.com/kubernetes/minikube) and
[BOSH Lite](https://bosh.io/docs/bosh-lite/) to run Kubernetes and BOSH, respectively, on your local
workstation as VMs using [VirtualBox](https://www.virtualbox.org/).

This has been tested to work with the following configurations:

```
$ cd ~/workspace/bosh-deployment

$ git show -s --oneline --no-decorate
8af402f Bumping uaa to version 64.0

$ minikube version
minikube version: v1.2.0
```

## Concepts

This project allows you to create and manage custom resources through Kubernetes which map to various
concepts in BOSH. The following assumes the reader is already fairly familiar with BOSH concepts, and
provides a high-level explanation of the kinds of custom resources offered, what they map to in the BOSH
world, and how they relate to one another:

### Director

The `Director` kind of resource provided by the `directors.bosh.akgupta.ca` CRD represents a BOSH
Director running somewhere in the world. It represents admin access to this Director, and more
specifically, to the UAA. This kind of resource should be created and managed by a Kubernetes cluster
administrator. Developers who are using the Kubernetes cluster can utilize any of the Directors by
requesting tenancy in a given Director via the [Team](#team) construct.

### Team

The `Team` kind of resource is provided by the `teams.bosh.akgupta.ca` CRD. Each team must reference
exactly one `Director`. A developer would create a `Team` in his or her namespace, referencing a
`Director`. Then, in theory, all subsequent BOSH custom resources created within that namespace would be
scoped to a dedicated tenant within that BOSH Director as represented by the namespace's `Team`. Currently,
creating one of these resources creates a client in the referenced Director's UAA and in theory it would
just have admin access within a BOSH Team generated for this purpose. There should be at most one `Team`
per namespace. Multiple `Team`s can refer to the same `Director`. `Team`s cannot be mutated. Deleting a `Team`
custom resource will delete the client from the corresponding BOSH Director's UAA (and the Kubernetes
`Secret` resource that is dynamically created to store the UAA client secret).

### Release

The `Release` kind of resource provided by the `releases.bosh.akgupta.ca` CRD represents BOSH releases
that can be uploaded to the Director. Creating one of these resources requires providing a URL for the
BOSH release. This release will be uploaded via the `Team` in the same namespace where the `Release`
resource has been created. The link between a `Release` and a `Team` is implicit by virtue of being in the
same namespace. `Release`s cannot be mutated. Deleting a `Release` custom resource will delete the release from
the corresponding BOSH Director.

### Stemcell

The `Stemcell` kind of resource provided by the `stemcells.bosh.akgupta.ca` CRD represents BOSH stemcells
that can be uploaded to the Director. Creating one of these resources requires providing a URL for the
BOSH stemcell. This stemcell will be uploaded via the `Team` in the same namespace where the `Stemcell`
resource has been created. The link between a `Stemcell` and a `Team` is implicit by virtue of being in the
same namespace.  `Stemcell`s cannot be mutated. Deleting a `Stemcell` custom resource will delete the stemcell
from the corresponding BOSH Director.

### VM Extension

The `VMExtension` kind of resource provided by the `vmextensions.bosh.akgupta.ca` CRD represents VM extensions
that traditionally live in a "Cloud Config". This "BOSH v3" API eschews the complex, monolithic "Cloud Config"
and treats VM Extensions as their own, first-class resource. These will be referenceable by name within
Instance Groups (not yet implemented). Creating one of these VM Extension resources requires simply providing
`cloud_properties`. This VM extension will be created via the `Team` in the same namespace where the `VMExtension`
resource has been created. The link between a `VMExtension` and a `Team` is implicit by virtue of being in the
same namespace.  `VMExtension`s cannot be mutated. Deleting a `VMExtension` custom resource will delete it from
from the corresponding BOSH Director.

### AZ

The `AZ` kind of resource provided by the `azs.bosh.akgupta.ca` CRD represents AZs (Availability Zones) that
traditionally live in a "Cloud Config". This "BOSH v3" API eschews the complex, monolithic "Cloud Config" and
treats AZs as their own, first-class resource. These will be referenceable by name within Instance Groups (not
yet implemented). Creating one of these AZ resources requires simply providing `cloud_properties`. This AZ will
be created via the `Team` in the same namespace where the `AZ` resource has been created. The link between an
`AZ` and a `Team` is implicit by virtue of being in the same namespace.  `AZ`s cannot be mutated. Deleting an
`AZ` custom resource will delete it from from the corresponding BOSH Director.

## Usage

### As an Administrator

Once you've installed this into your Kubernetes cluster, the controller will be running in a BOSH system
namespace (defaults to `bosh-system`). Any `Director` resources you create must be created in this
namespace. You will also need to create `Secret` resources to contain the UAA admin client secrets to make
your `Director` resources usable -- those too must reside in this namespace. When consumer request `Team`s,
the controller will dynamically generate `Secret` resources to contain the UAA client secrets corresponding
to each `Team` -- those will be generated within this namespace and should not be tampered with.

The following `Director` YAML skeleton includes instructions for how you would find the right values to
populate:

```
apiVersion: "bosh.akgupta.ca/v1"
kind: Director
metadata:
  name: <DIRECTOR_NAME> # any name you want
  namespace: <BOSH_SYSTEM_NAMESPACE> # must be the same namespace where the controller was deployed
spec:
  url: "https://<BOSH_ADDRESS>"
  ca_cert: # `bosh int --path /director_ssl/ca creds.yml` where `creds.yml` is the vars file generated
           # when creating BOSH
  uaa_url: # "https://<BOSH_ADDRESS>:8443" if you've deployed BOSH via the official docs
  uaa_client: # "uaa_admin" if you've deployed BOSH via the official docs without major tweaks
  uaa_client_secret: <UAA_SECRET_NAME> # any name you want
  uaa_ca_cert: # `bosh int --path /uaa_ssl/ca creds.yml` where `creds.yml` is the vars file generated
               # when creating BOSH
    
```

Create a `Secret` in the BOSH system namespace with name matching `<UAA_SECRET_NAME>`:

```
$ kubectl create secret generic <UAA_SECRET_NAME> \
  --from-literal=secret="$(bosh int --path /uaa_admin_client_secret <PATH_TO_CREDS_YML>)" \
  --namespace=<BOSH_SYSTEM_NAMESPACE> # must be the same namespace where the controller was deployed
```

When giving individual developers access to use the Kubernetes cluster, you will likely want to only give
them permissions to manage resources within a given namespace. They should generally _not_ have access to
resources in the BOSH system namespace, but if you'd like them to be able to request tenants within the
configured `Director`s, they should have the ability to get and list `Director` resources in the BOSH system
namespace.

### As a Developer

As a developer or user of Kubernetes, each `Director` resource can be regarded as a service offering and you
would like tenancy within one of the Directors to use it to do BOSH-y thing. Start by creating a `Team` in
your namespace. For example:

```
apiVersion: "bosh.akgupta.ca/v1"
kind: Team
metadata:
  name: <SOME_TEAM_NAME>
  namespace: <MY_DEV_NAMESPACE>
spec:
  director: <DIRECTOR_NAME>
```

You simply need to ensure that `<DIRECTOR_NAME>` matches one of the available `Director` resources offered by
the cluster administrator.

Once you have a `Team` in your namespace you can manage BOSH resources like stemcells and releases by
creating corresponding custom resources in your namespace. See below on detailed specifications for each
custom resource.

## Specification

### Director

```
kind: Director
spec:
  url: # URL of the BOSH Director
  ca_cert: # CA certificate for the controller to trust when communicating with the BOSH Director
  uaa_url: # URL for the BOSH Director's UAA
  uaa_client: # Name of the UAA admin client
  uaa_client_secret: # Name of the Kubernetes Secret resource where you'll store the client secret of the
                     # UAA admin client. The secret value must be stored in the "secret" key within the
                     # data stored in the Secret resource.
  uaa_ca_cert: # CA certificate for the controller to trust when communicating with the BOSH Director's UAA
```

You can inspect this resource and expect output like the following:

```
$ kubectl get director --all-namespaces
NAMESPACE     NAME         URL                    UAA CLIENT
<BOSH_SYSTEM_NAMESPACE> # must be the same namespace where the controller was deployed   vbox-admin   https://192.168.50.6   uaa_admin
```

### Team

```
kind: Team
spec:
  director: # Name of a Director custom resource
```

You can inspect this resource and expect output like the following:

```
$ kubectl get team --all-namespaces
NAMESPACE   NAME   DIRECTOR     AVAILABLE   WARNING
test        test   vbox-admin   true
```

The `AVAILABLE` column will show `false` if the UAA client for the team has not been successfully created.
The `WARNING` column will display a warning if you have mutated the `Team` spec after initial creation. The
`DIRECTOR` column displays the originally provided value for `spec.director` and this is the value that
this team will continue to use. If you do attempt to mutate the `Team` resource, you can see your
(ignored) user-provided value with the `-o wide` flag:

```
$ kubectl get team --all-namespaces -owide
NAMESPACE   NAME   DIRECTOR     AVAILABLE   WARNING   USER-PROVIDED DIRECTOR
test        test   vbox-admin   true                  vbox-admin
```

If we attempt to mutate the `spec.director` property, here's what we will see:

```
$ kubectl get team --all-namespaces -owide
NAMESPACE   NAME   DIRECTOR     AVAILABLE   WARNING                                              USER-PROVIDED DIRECTOR
test        test   vbox-admin   true        API resource has been mutated; all changes ignored   bad-new-director-name
```

### Release

```
kind: Release
spec:
  releaseName: # Name of the BOSH release
  version: # Version of the BOSH release
  url: # URL where the Director will fetch the BOSH release artifact from
  sha1: # SHA1 checksum of the BOSH release artifact for the Director to confirm
```

You can inspect this resource and expect output like the following:

```
$ kubectl get release --all-namespaces
NAMESPACE   NAME              RELEASE NAME   VERSION   AVAILABLE   WARNING
test        zookeeper-0.0.9   zookeeper      0.0.9     false
```

This is what you'll see before the Director has completed fetching the release. After it has, you'll see:

```
$ kubectl get release --all-namespaces
NAMESPACE   NAME              RELEASE NAME   VERSION   AVAILABLE   WARNING
test        zookeeper-0.0.9   zookeeper      0.0.9     true
```

The `AVAILABLE` column will show `false` if the Director has not been successfully fetched the release.
The `WARNING` column will display a warning if you have mutated the `Release` spec after initial creation. The
`RELEASE NAME` and `VERSION` columns display the originally provided values and these are the values that will
continue to be used. If you do attempt to mutate the `Release` resource, you can see your (ignored) user-provided
values with the `-o wide` flag, along with the original values and (ignored, possibly-muted) subsequent
user-provided values for URL and SHA1.

### Stemcell

```
kind: Stemcell
spec:
  stemcellName: # Name of the BOSH stemcell
  version: # Version of the BOSH stemcell
  url: # URL where the Director will fetch the BOSH stemcell artifact from
  sha1: # SHA1 checksum of the BOSH stemcell artifact for the Director to confirm
```

The behaviour of `kubectl get stemcell` is essentially identical to the behaviour for `kubectl get release`
described in the previous sub-section.

### VM Extension

```
kind: VMExtension
spec:
  cloud_properties:
    # YAML or JSON of CPI-specific Cloud Properties for VM extensions
```

You can inspect this resource and expect output like the following:

```
$ kubectl get vmextension --all-namespaces
NAMESPACE   NAME                AVAILABLE   WARNING
test        port-tcp-443-8443   false
```

This is what you'll see before the Director has completed fetching the release. After it has, you'll see:

```
$ kubectl get release --all-namespaces
NAMESPACE   NAME                AVAILABLE   WARNING
test        port-tcp-443-8443   true
```

The `AVAILABLE` column will show `false` if the cloud-type config hasn't been successfully posted to the Director.
The `WARNING` column will display a warning if you have mutated the `VMExtension` spec after initial creation.

### AZ

```
kind: AZ
spec:
  cloud_properties:
    # YAML or JSON of CPI-specific Cloud Properties for AZs
```

The behaviour of `kubectl get az` is essentially identical to the behaviour for `kubectl get vmextension`
described in the previous sub-section.

## TODO

1. Implement [more BOSH resources](remaining_bosh_resources.txt)
1. Make better secrets
1. Tests 😂
1. Reproducible builds

## Issues

### CRDs

- Would like to set OwnerReferences to resources in other namespaces or cluster-scoped resources so that
child resources can be automatically garbage-collected.
- Would like to be able to easily enforce validations (e.g. some resource is a singleton and there can
only be one of them per namespace).
- More generally, would more flexible, in-code validations for custom resources without the heavyweight
need to implement webhooks.
- Would like to enforce immutability of some/all fields in a custom resource spec.

Larger architectural concerns and concerns related to the developer experience for creating CRDs are outside
the scope of this README.

### Kubebuilder

- Creating cluster-scoped (as opposed to namespace-scoped) CRDs doesn't seem to work well, e.g.
`kubebuilder create api --namespaced=false` doesn't seem to do the expected thing, and if it did, the 
generated tests just fail out of the box.
- The generated [samples](config/samples) are unusable and need to be modified. There doesn't seem to be
much value from the files generated here by Kubebuilder.
- The Makefile could be greatly streamlined. See
[here](https://miro.com/app/board/o9J_kxIYNts=/?moveToWidget=3074457346709258907).
- The directories and files in the `config` directory seem inconstent and don't work in a variety of ways.
- Status subresources should be enabled out of the box for new CRDs, or at least enabled easily without
repeatedly creating Kustomize overlays for each CRD.

### BOSH

- UAA clients require `bosh.admin` scope to delete releases and stemcells, even though they just need
`bosh.stemcells.upload` and `bosh.releases.upload` to upload them. Better fine-grained permissions in BOSH
would be very nice.
- In the same vein, scoping BOSH resources to teams would be nice. This project provides a facade where it
appears each Kubernetes namespace has a separate BOSH team and custom resources representing BOSH resources
are created in individual namespaces where they appear scoped to an individual BOSH team, but they are in
reality global resources in BOSH and prone to collision.
- The `config cmdconf.Config` argument to `director.Factory#New` in the
[`BOSH CLI codebase`](https://github.com/cloudfoundry/bosh-cli/blob/7850ac985726c614f8ecba726ae8c0d17b08ad7f/director/factory.go#L29)
is unused and should be removed.

### Minikube

- If I want to upgrade my Minikube to use Kubernetes 1.15, how do I do this? Does it entail a full teardown
and restart of my Minikube cluster, including all its state?

## Development

### Requirements

You will need to have some prerequisites installed such as `go`, `make`, `kustomize`, `kubectl`, `docker`,
and `kubebuilder` installed. You should have the `bosh` CLI installed so you can target BOSH Directors
directly and manually test that the right resources/tasks are being created/run in BOSH. This project
manages UAA clients associated with BOSH Directors, so it can be useful to have the `uaac` CLI installed as
well.

### Makefile

Most of the important tasks for your development lifecycle are in the [`Makefile`](Makefile). An overview
of the tasks in the Makefile and their relationships to one another is available
[here](https://miro.com/app/board/o9J_kxIYNts=/?moveToWidget=3074457346709258907).

### Creating a new API

To create new CRDs and reconciliation controllers for managing BOSH resources, use

```
$ kubebuilder create api --controller --example=false --group=bosh --kind=<SomeKind> --resource --version=v1

```

You will need to update code and YAML templates in the `api`, `config`, and `controllers` subdirectories.
The [Kubebuilder Book](https://book.kubebuilder.io/introduction.html) is a good, albeit rough, resource for
learning some of the requisite concepts in a practical way. See especially the sections on
[Designing an API](https://book.kubebuilder.io/cronjob-tutorial/api-design.html), 
[Implementing a controller](https://book.kubebuilder.io/cronjob-tutorial/controller-implementation.html),
and [Using Finalizers](https://book.kubebuilder.io/reference/using-finalizers.html).

NOTE: Replace the `--controller` flag with `--controller=false` or the `--resource` flag with `--resource=false`
if you don't need a reconciliation controller or don't need a CRD, respectively. For example, the `Director`
kind of resource does not need reconciliation, it just needs to be referenceable from `Team`s, so it was
created with `--controller=false` in the above `kubebuilder` command.

### Build, Run, and Test

- `make` generates code and YAML, and builds an executable locally, ensuring that the code compiles.
- `make run` applies CRD YAML configuration files to the targetted Kubernetes cluster and runs
the controllers as a local process interacting with the Kubernetes API.
- `kubectl apply -f <file>` some custom resources and use `bosh` and `uaac` to check that the right things
are happening. Consider trying out the [samples](config/samples).
- `kubectl get bosh --all-namespaces` gives a view from the Kubernetes API perspective, which is meant to
respresent the "BOSH v3 API" this project is intended to explore as a concept

### Publish, Deploy, and Test

- `make image repo` to publish a Docker image containing the controllers.
- `make install` to apply the same CRD YAML configurations as in `make run`, as well as YAML
configurations for deploying the controllers to the Kubernetes cluster with the published Docker image and 
authorizing them via RBAC policies to function properly.
- Test things out same as above.

### Document

- Update the relevant sections in this README and add a workable example in `config/samples`.

## License

[Apache license 2.0](LICENSE) © Amit Kumar Gupta.
