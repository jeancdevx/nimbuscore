package operator

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (ws *Workspace) DeepCopyObject() runtime.Object {
	return ws.DeepCopy()
}

func (ws *Workspace) DeepCopy() *Workspace {
	out := *ws
	out.TypeMeta = ws.TypeMeta
	out.ObjectMeta = *ws.ObjectMeta.DeepCopy()

	out.Spec = *ws.Spec.DeepCopy()
	out.Status = *ws.Status.DeepCopy()

	return &out
}

func (ws *WorkspaceSpec) DeepCopy() *WorkspaceSpec {
	out := *ws

	if out.Ingress.Ports != nil {
		ports := make([]int, len(ws.Ingress.Ports))
		copy(ports, ws.Ingress.Ports)
		out.Ingress.Ports = ports
	}

	return &out
}

func (ws *WorkspaceStatus) DeepCopy() *WorkspaceStatus {
	out := *ws

	if ws.Conditions != nil {
		conds := make([]metav1.Condition, len(ws.Conditions))
		copy(conds, ws.Conditions)
		out.Conditions = conds
	}

	return &out
}

func (wl *WorkspaceList) DeepCopyObject() runtime.Object {
	return wl.DeepCopy()
}

func (wl *WorkspaceList) DeepCopy() *WorkspaceList {
	out := *wl
	out.TypeMeta = wl.TypeMeta
	out.ListMeta = *wl.ListMeta.DeepCopy()

	if wl.Items != nil {
		items := make([]Workspace, len(wl.Items))
		for i := range wl.Items {
			items[i] = *wl.Items[i].DeepCopy()
		}
		out.Items = items
	}

	return &out
}
