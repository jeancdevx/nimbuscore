package controller

import (
	"context"
	"fmt"
	"log/slog"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nimbuscorev1alpha1 "github.com/nimbuscore/pkg/operator"
)

const prebuildFinalizer = "nimbuscore.io/prebuild-finalizer"

type PrebuildReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	OutputRegistry string
}

func RegisterPrebuildController(mgr ctrl.Manager, outputRegistry string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nimbuscorev1alpha1.Prebuild{}).
		Owns(&batchv1.Job{}).
		Complete(&PrebuildReconciler{
			Client:         mgr.GetClient(),
			Scheme:         mgr.GetScheme(),
			OutputRegistry: outputRegistry,
		})
}

func (r *PrebuildReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := slog.With("prebuild", req.NamespacedName)
	logger.Info("reconciling prebuild")

	var prebuild nimbuscorev1alpha1.Prebuild
	if err := r.Get(ctx, req.NamespacedName, &prebuild); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if !prebuild.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&prebuild, prebuildFinalizer) {
			controllerutil.RemoveFinalizer(&prebuild, prebuildFinalizer)
			if err := r.Update(ctx, &prebuild); err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&prebuild, prebuildFinalizer) {
		controllerutil.AddFinalizer(&prebuild, prebuildFinalizer)
		if err := r.Update(ctx, &prebuild); err != nil {
			return reconcile.Result{}, err
		}
	}

	switch prebuild.Status.Phase {
	case "", "pending":
		return r.handlePending(ctx, &prebuild)
	case "building":
		return r.handleBuilding(ctx, &prebuild)
	case "available", "failed":
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

func (r *PrebuildReconciler) handlePending(ctx context.Context, pb *nimbuscorev1alpha1.Prebuild) (reconcile.Result, error) {
	slog.Info("starting prebuild build", "prebuild", pb.Name)

	job := r.buildJob(pb)
	if err := r.Create(ctx, job); err != nil && !apierrors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	}

	pb.Status.Phase = nimbuscorev1alpha1.PrebuildPhaseBuilding
	pb.Status.Message = "building prebuild image"
	if err := r.Status().Update(ctx, pb); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

func (r *PrebuildReconciler) handleBuilding(ctx context.Context, pb *nimbuscorev1alpha1.Prebuild) (reconcile.Result, error) {
	var job batchv1.Job
	if err := r.Get(ctx, types.NamespacedName{Name: pb.Name, Namespace: pb.Namespace}, &job); err != nil {
		if apierrors.IsNotFound(err) {
			pb.Status.Phase = nimbuscorev1alpha1.PrebuildPhaseFailed
			pb.Status.Message = "build job not found"
			r.Status().Update(ctx, pb)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			pb.Status.Phase = nimbuscorev1alpha1.PrebuildPhaseAvailable
			pb.Status.Message = "prebuild image available"
			pb.Status.PodName = ""
			r.Status().Update(ctx, pb)
			return reconcile.Result{}, nil
		}
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			pb.Status.Phase = nimbuscorev1alpha1.PrebuildPhaseFailed
			pb.Status.Message = fmt.Sprintf("build failed: %s", c.Message)
			r.Status().Update(ctx, pb)
			return reconcile.Result{}, nil
		}
	}

	return reconcile.Result{Requeue: true}, nil
}

func (r *PrebuildReconciler) buildJob(pb *nimbuscorev1alpha1.Prebuild) *batchv1.Job {
	outputImage := fmt.Sprintf("%s/%s:%s", r.OutputRegistry, pb.Spec.OutputImage, pb.Name)
	devcontainerPath := pb.Spec.DevcontainerPath
	if devcontainerPath == "" {
		devcontainerPath = ".devcontainer/devcontainer.json"
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pb.Name,
			Namespace: pb.Namespace,
			Labels: map[string]string{
				nimbuscorev1alpha1.LabelPrebuildID: pb.Name,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: int32Ptr(2),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						nimbuscorev1alpha1.LabelPrebuildID: pb.Name,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "builder",
							Image: pb.Spec.Image,
							Command: []string{"sh", "-c"},
							Args: []string{
								fmt.Sprintf(`git clone --branch %s %s /workspace && \
									cd /workspace && \
									cp %s .devcontainer.json && \
									echo "Prebuild complete for %s"`, pb.Spec.Branch, pb.Spec.RepoURL, devcontainerPath, outputImage),
							},
							Env: []corev1.EnvVar{
								{Name: "PREBUILD_ID", Value: pb.Name},
								{Name: "OUTPUT_IMAGE", Value: outputImage},
							},
						},
					},
				},
			},
		},
	}

	return job
}

func int32Ptr(i int32) *int32 {
	return &i
}
