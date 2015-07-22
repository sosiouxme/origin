package client

// The purpose of this diagnostic is to detect nodes that are out of commission
// (which may affect the ability to schedule pods) for user awareness.

import (
	"errors"
	"fmt"

	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kclient "github.com/GoogleCloudPlatform/kubernetes/pkg/client"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/fields"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/openshift/origin/pkg/diagnostics/log"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

const (
	clientErrorGettingNodes = `Client error while retrieving node records. Client retrieved records
during discovery, so this is likely to be a transient error. Try running
diagnostics again. If this message persists, there may be a permissions
problem with getting node records. The error was:

(%T) %[1]v`

	nodeNotReady = `Node {{.node}} is defined but is not marked as ready.
Ready status is {{.status}} because "{{.reason}}"
If the node is not intentionally disabled, check that the master can
reach the node hostname for a health check and the node is checking in
to the master with the same hostname.

While in this state, pods should not be scheduled to deploy on the node,
and any existing scheduled pods will be considered failed and removed.
`

	nodeNotSched = `Node {{.node}} is ready but is marked Unschedulable.
This is usually set manually for administrative reasons.
An administrator can mark the node schedulable with:
    oadm manage-node {{.node}} --schedulable=true

While in this state, pods should not be scheduled to deploy on the node.
Existing pods will continue to run until completed or evacuated (see
other options for 'oadm manage-node').
`
)

// NodeDefinitions
type NodeDefinition struct {
	KubeClient *kclient.Client

	Log *log.Logger
}

func (d NodeDefinition) Description() string {
	return "Check node records on master"
}

func (d NodeDefinition) CanRun() (bool, error) {
	if d.KubeClient == nil {
		return false, errors.New("must have kube client")
	}
	if _, err := d.KubeClient.Nodes().List(labels.LabelSelector{}, fields.Everything()); err != nil {
		// TODO check for 403 to return: "Client does not have cluster-admin access and cannot see node records"

		return false, types.NewDiagnosticError("clGetNodesFailed", fmt.Sprintf(clientErrorGettingNodes, err), err)
	}

	return true, nil
}

func (d NodeDefinition) Check() *types.DiagnosticResult {
	r := &types.DiagnosticResult{}

	nodes, err := d.KubeClient.Nodes().List(labels.LabelSelector{}, fields.Everything())
	if err != nil {
		return r.Error(types.NewDiagnosticError("clGetNodesFailed",
			fmt.Sprintf(clientErrorGettingNodes, err), err))
	}

	anyNodesAvail := false
	for _, node := range nodes.Items {
		var ready *kapi.NodeCondition
		for i, condition := range node.Status.Conditions {
			switch condition.Type {
			// Each condition appears only once. Currently there's only one... used to be more
			case kapi.NodeReady:
				ready = &node.Status.Conditions[i]
			}
		}

		if ready == nil || ready.Status != kapi.ConditionTrue {
			// instead of building this, simply use the node object directly
			templateData := map[string]interface{}{}
			templateData["node"] = node.Name
			if ready == nil {
				templateData["status"] = "None"
				templateData["reason"] = "There is no readiness record."
			} else {
				templateData["status"] = ready.Status
				templateData["reason"] = ready.Reason
			}

			r.Warn(types.NewDiagnosticErrorFromTemplate("clNodeNotReady", nodeNotReady, templateData))
		} else if node.Spec.Unschedulable {
			r.Warn(types.NewDiagnosticErrorFromTemplate("clNodeNotSched", nodeNotSched, map[string]interface{}{"node": node.Name}))
		} else {
			anyNodesAvail = true
		}
	}
	if !anyNodesAvail {
		r.Error(types.NewDiagnosticError("clNoAvailNodes", "There were no nodes availabable for OpenShift to use.", nil))
	}

	return r
}
