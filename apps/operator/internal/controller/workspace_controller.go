package controller

import (
	"context"
	"fmt"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nimbuscorev1alpha1 "github.com/nimbuscore/pkg/operator"
)

const workspaceFinalizer = "nimbuscore.io/workspace-finalizer"

type WorkspaceReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	IngressHost   string
	TLSSecretName string
}

func RegisterWorkspaceController(mgr ctrl.Manager, ingressHost, tlsSecret string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nimbuscorev1alpha1.Workspace{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Complete(&WorkspaceReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			IngressHost:   ingressHost,
			TLSSecretName: tlsSecret,
		})
}

func (r *WorkspaceReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := slog.With("workspace", req.NamespacedName)
	logger.Info("reconciling workspace")

	var workspace nimbuscorev1alpha1.Workspace
	if err := r.Get(ctx, req.NamespacedName, &workspace); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if workspace.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&workspace, workspaceFinalizer) {
			controllerutil.AddFinalizer(&workspace, workspaceFinalizer)
			if err := r.Update(ctx, &workspace); err != nil {
				return reconcile.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(&workspace, workspaceFinalizer) {
			if err := r.cleanupWorkspace(ctx, &workspace); err != nil {
				return reconcile.Result{}, err
			}
			controllerutil.RemoveFinalizer(&workspace, workspaceFinalizer)
			if err := r.Update(ctx, &workspace); err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	switch workspace.Status.Phase {
	case "", "pending":
		return r.handlePending(ctx, &workspace)
	case "building":
		return r.handleBuilding(ctx, &workspace)
	case "running":
		return r.handleRunning(ctx, &workspace)
	case "stopping":
		return r.handleStopping(ctx, &workspace)
	default:
		return reconcile.Result{}, nil
	}
}

func (r *WorkspaceReconciler) handlePending(ctx context.Context, ws *nimbuscorev1alpha1.Workspace) (reconcile.Result, error) {
	slog.Info("creating namespace for workspace", "workspace", ws.Name)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ws.Name,
			Labels: map[string]string{
				nimbuscorev1alpha1.LabelWorkspaceID: ws.Name,
				nimbuscorev1alpha1.LabelUserID:      ws.Spec.UserID,
			},
		},
	}

	if err := r.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	}

	ws.Status.Phase = "building"
	ws.Status.Message = "namespace created, provisioning resources"
	if err := r.Status().Update(ctx, ws); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

func (r *WorkspaceReconciler) handleBuilding(ctx context.Context, ws *nimbuscorev1alpha1.Workspace) (reconcile.Result, error) {
	slog.Info("provisioning workspace resources", "workspace", ws.Name)

	pvc := r.buildPVC(ws)
	if err := r.Create(ctx, pvc); err != nil && !apierrors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	}

	svc := r.buildService(ws)
	if err := r.Create(ctx, svc); err != nil && !apierrors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	}

	pod := r.buildPod(ws)
	if err := r.Create(ctx, pod); err != nil && !apierrors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	}

	ing := r.buildIngress(ws)
	if err := r.Create(ctx, ing); err != nil && !apierrors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	}

	portURLs := r.buildPortIngresses(ctx, ws)

	ws.Status.Phase = "running"
	ws.Status.Message = "workspace is running"
	ws.Status.PodName = pod.Name
	ws.Status.URL = fmt.Sprintf("http://%s.%s", ws.Name, r.IngressHost)
	ws.Status.PortURLs = portURLs
	if err := r.Status().Update(ctx, ws); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *WorkspaceReconciler) handleRunning(ctx context.Context, ws *nimbuscorev1alpha1.Workspace) (reconcile.Result, error) {
	var pod corev1.Pod
	if err := r.Get(ctx, types.NamespacedName{Name: ws.Name, Namespace: ws.Name}, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			ws.Status.Phase = "error"
			ws.Status.Message = "pod not found"
			r.Status().Update(ctx, ws)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	ws.Status.PodStatus = pod.Status.Phase
	if err := r.Status().Update(ctx, ws); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

func (r *WorkspaceReconciler) handleStopping(ctx context.Context, ws *nimbuscorev1alpha1.Workspace) (reconcile.Result, error) {
	slog.Info("stopping workspace", "workspace", ws.Name)

	var pod corev1.Pod
	if err := r.Get(ctx, types.NamespacedName{Name: ws.Name, Namespace: ws.Name}, &pod); err == nil {
		if err := r.Delete(ctx, &pod); err != nil {
			return reconcile.Result{}, err
		}
	}

	ws.Status.Phase = "stopped"
	ws.Status.Message = "workspace stopped"
	ws.Status.PodName = ""
	if err := r.Status().Update(ctx, ws); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *WorkspaceReconciler) cleanupWorkspace(ctx context.Context, ws *nimbuscorev1alpha1.Workspace) error {
	slog.Info("cleaning up workspace", "workspace", ws.Name)

	var ns corev1.Namespace
	if err := r.Get(ctx, types.NamespacedName{Name: ws.Name}, &ns); err == nil {
		if err := r.Delete(ctx, &ns); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (r *WorkspaceReconciler) buildPVC(ws *nimbuscorev1alpha1.Workspace) *corev1.PersistentVolumeClaim {
	var storageClass *string
	if ws.Spec.Storage.StorageClass != "" {
		storageClass = &ws.Spec.Storage.StorageClass
	}
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ws.Name + "-home",
			Namespace: ws.Name,
			Labels:    map[string]string{nimbuscorev1alpha1.LabelWorkspaceID: ws.Name},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: storageClass,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(ws.Spec.Resources.Disk),
				},
			},
		},
	}
}

func (r *WorkspaceReconciler) buildService(ws *nimbuscorev1alpha1.Workspace) *corev1.Service {
	ports := []corev1.ServicePort{
		{Name: "code-server", Port: 80, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP},
	}

	for i, p := range ws.Spec.Ingress.Ports {
		proto := corev1.ProtocolTCP
		if p.Protocol == "udp" {
			proto = corev1.ProtocolUDP
		}
		portName := fmt.Sprintf("port-%d", i)
		if p.Subdomain != "" {
			portName = p.Subdomain
		}
		ports = append(ports, corev1.ServicePort{
			Name:       portName,
			Port:       int32(p.Port),
			TargetPort: intstr.FromInt(p.Port),
			Protocol:   proto,
		})
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ws.Name,
			Namespace: ws.Name,
			Labels:    map[string]string{nimbuscorev1alpha1.LabelWorkspaceID: ws.Name},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{nimbuscorev1alpha1.LabelWorkspaceID: ws.Name},
			Ports:    ports,
		},
	}
}

func (r *WorkspaceReconciler) buildPortIngresses(ctx context.Context, ws *nimbuscorev1alpha1.Workspace) map[int]string {
	portURLs := make(map[int]string)

	for _, rule := range ws.Spec.Ingress.Ports {
		if rule.Subdomain == "" {
			continue
		}

		host := fmt.Sprintf("%s-%s.%s", rule.Subdomain, ws.Name, r.IngressHost)
		portURLs[rule.Port] = fmt.Sprintf("http://%s", host)

		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", ws.Name, rule.Subdomain),
				Namespace: ws.Name,
				Labels:    map[string]string{nimbuscorev1alpha1.LabelWorkspaceID: ws.Name},
				Annotations: map[string]string{
					"nginx.ingress.kubernetes.io/proxy-body-size":    "0",
					"nginx.ingress.kubernetes.io/proxy-read-timeout": "3600",
					"nginx.ingress.kubernetes.io/proxy-send-timeout": "3600",
				},
			},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{
						Host: host,
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
										PathType: pathTypePtr(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: ws.Name,
												Port: networkingv1.ServiceBackendPort{
													Number: int32(rule.Port),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		if r.TLSSecretName != "" {
			ing.Spec.TLS = []networkingv1.IngressTLS{
				{
					Hosts:      []string{host},
					SecretName: r.TLSSecretName,
				},
			}
		}

		if err := r.Create(ctx, ing); err != nil && !apierrors.IsAlreadyExists(err) {
			slog.Warn("failed to create port ingress", "workspace", ws.Name, "port", rule.Port, "error", err)
		}
	}

	return portURLs
}

func (r *WorkspaceReconciler) buildIngress(ws *nimbuscorev1alpha1.Workspace) *networkingv1.Ingress {
	host := fmt.Sprintf("%s.%s", ws.Name, r.IngressHost)

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ws.Name,
			Namespace: ws.Name,
			Labels:    map[string]string{nimbuscorev1alpha1.LabelWorkspaceID: ws.Name},
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/proxy-body-size": "0",
				"nginx.ingress.kubernetes.io/proxy-read-timeout": "3600",
				"nginx.ingress.kubernetes.io/proxy-send-timeout": "3600",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: pathTypePtr(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: ws.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if r.TLSSecretName != "" {
		ing.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      []string{host},
				SecretName: r.TLSSecretName,
			},
		}
	}

	return ing
}

func (r *WorkspaceReconciler) buildPod(ws *nimbuscorev1alpha1.Workspace) *corev1.Pod {
	cpuQty := resource.MustParse(ws.Spec.Resources.CPU)
	memQty := resource.MustParse(ws.Spec.Resources.Memory)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ws.Name,
			Namespace: ws.Name,
			Labels: map[string]string{
				nimbuscorev1alpha1.LabelWorkspaceID: ws.Name,
				nimbuscorev1alpha1.LabelUserID:      ws.Spec.UserID,
			},
			Annotations: map[string]string{
				nimbuscorev1alpha1.AnnotationRepoURL: ws.Spec.RepoURL,
				nimbuscorev1alpha1.AnnotationBranch:  ws.Spec.Branch,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "code-server",
					Image: ws.Spec.Image,
					Ports: []corev1.ContainerPort{{ContainerPort: 8080}},
					Env: []corev1.EnvVar{
						{Name: "NAMESPACE", Value: ws.Name},
						{Name: "WORKSPACE_ID", Value: ws.Name},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    cpuQty,
							corev1.ResourceMemory: memQty,
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    cpuQty,
							corev1.ResourceMemory: memQty,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "home", MountPath: "/home/coder"},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "home",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: ws.Name + "-home",
						},
					},
				},
			},
		},
	}

	if ws.Spec.RepoURL != "" {
		pod.Spec.InitContainers = []corev1.Container{
			{
				Name:    "clone-repo",
				Image:   "alpine/git:latest",
				Command: []string{"sh", "-c"},
				Args: []string{
					"git clone --branch " + ws.Spec.Branch + " " + ws.Spec.RepoURL + " /home/coder/project",
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "home", MountPath: "/home/coder"},
				},
			},
		}
	}

	return pod
}

func pathTypePtr(t networkingv1.PathType) *networkingv1.PathType {
	return &t
}
