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

	if ws.Ingress.Ports != nil {
		ports := make([]PortRule, len(ws.Ingress.Ports))
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

	out.PortURLs = make(map[int]string, len(ws.PortURLs))
	for k, v := range ws.PortURLs {
		out.PortURLs[k] = v
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

func (pb *Prebuild) DeepCopyObject() runtime.Object {
	return pb.DeepCopy()
}

func (pb *Prebuild) DeepCopy() *Prebuild {
	out := *pb
	out.TypeMeta = pb.TypeMeta
	out.ObjectMeta = *pb.ObjectMeta.DeepCopy()
	out.Spec = pb.Spec
	out.Status.Conditions = make([]metav1.Condition, len(pb.Status.Conditions))
	copy(out.Status.Conditions, pb.Status.Conditions)
	return &out
}

func (pl *PrebuildList) DeepCopyObject() runtime.Object {
	return pl.DeepCopy()
}

func (pl *PrebuildList) DeepCopy() *PrebuildList {
	out := *pl
	out.TypeMeta = pl.TypeMeta
	out.ListMeta = *pl.ListMeta.DeepCopy()

	if pl.Items != nil {
		items := make([]Prebuild, len(pl.Items))
		for i := range pl.Items {
			items[i] = *pl.Items[i].DeepCopy()
		}
		out.Items = items
	}

	return &out
}
