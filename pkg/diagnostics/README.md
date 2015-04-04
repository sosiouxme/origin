OpenShift v3 Diagnostics
========================

This is a tool to help administrators and users resolve common problems
that occur with OpenShift v3 deployments. It is currently (March 2015)
under heavy development in parallel with the OpenShift Origin project.

The goals of the diagnostics tool are summarized in this
[Trello card](https://trello.com/c/LdUogKuN/379-openshift-v3-diagnostics)
(account required). Eventually the functionality may be included more
directly in OpenShift itself, but initially it will be a separate tool
that analyzes OpenShift as it finds it, whether from the perspective
of an OpenShift client or on an OpenShift host.

Expected environment
====================

OpenShift can be deployed in many ways: built from source, included
in a VM image, in a Docker image, or as enterprise RPMs. When running
on OpenShift hosts, diagnostics will initially focus on detecting and
diagnosing the systemd services as provided in enterprise RPMs.
Detecting or specifying other deployment methods should come later.

The user may only have access as an ordinary user, as a cluster-admin
user, or may be root on a host where OpenShift master or node services
are operating. The diagnostics will attempt to use as much access as
the user has available.

`diagnostics` expects to find an `openshift` binary as well as `osc` in
order to make client calls. It searches your path for these or you
may specify where to find them. It also looks for a .kubeconfig (until
a newer configuration scheme is introduced soon).

Building diagnostics
====================

Eventually there will be regular releases of this tool but for now
it can be built from source. First you need to have
[golang installed](https://golang.org/doc/install). Then:

    git clone https://github.com/sosiouxme/openshift-extras -b enterprise-3.0-beta2
    cd openshift-extras/diagnostics
    make

This will create the "diagnostics" binary. Then you can simply run it:

    ./diagnostics help

This has not been tried on Windows and likely does not actually work
there, but Cygwin users are welcome to give it a try and let us know
how to improve it if needed.
