package operator

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: APIVersion}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion, &Workspace{}, &WorkspaceList{}, &Prebuild{}, &PrebuildList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

type PortRule struct {
	Port      int    `json:"port"`
	Protocol  string `json:"protocol,omitempty"`
	Subdomain string `json:"subdomain,omitempty"`
}

type WorkspaceSpec struct {
	UserID  string `json:"userId"`
	TeamID  string `json:"teamId,omitempty"`
	Image   string `json:"image"`
	RepoURL string `json:"repoUrl,omitempty"`
	Branch  string `json:"branch,omitempty"`

	Resources struct {
		CPU    string `json:"cpu"`
		Memory string `json:"memory"`
		Disk   string `json:"disk"`
		GPU    int    `json:"gpu,omitempty"`
	} `json:"resources"`

	Ingress struct {
		Enabled bool       `json:"enabled"`
		Host    string     `json:"host,omitempty"`
		Ports   []PortRule `json:"ports,omitempty"`
	} `json:"ingress"`

	Storage struct {
		Size         string `json:"size"`
		StorageClass string `json:"storageClass,omitempty"`
	} `json:"storage"`
}

type WorkspaceStatus struct {
	Phase      string             `json:"phase"`
	Message    string             `json:"message,omitempty"`
	PodName    string             `json:"podName,omitempty"`
	PodStatus  corev1.PodPhase    `json:"podStatus,omitempty"`
	URL        string             `json:"url,omitempty"`
	PortURLs   map[int]string     `json:"portUrls,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceSpec   `json:"spec,omitempty"`
	Status WorkspaceStatus `json:"status,omitempty"`
}

type WorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Workspace `json:"items"`
}

type PrebuildPhase string

const (
	PrebuildPhasePending   PrebuildPhase = "pending"
	PrebuildPhaseBuilding  PrebuildPhase = "building"
	PrebuildPhaseAvailable PrebuildPhase = "available"
	PrebuildPhaseFailed    PrebuildPhase = "failed"
)

type PrebuildSpec struct {
	UserID           string `json:"userId"`
	TeamID           string `json:"teamId,omitempty"`
	Image            string `json:"image"`
	RepoURL          string `json:"repoUrl"`
	Branch           string `json:"branch"`
	DevcontainerPath string `json:"devcontainerPath,omitempty"`
	OutputImage      string `json:"outputImage"`
}

type PrebuildStatus struct {
	Phase      PrebuildPhase      `json:"phase"`
	Message    string             `json:"message,omitempty"`
	PodName    string             `json:"podName,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type Prebuild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrebuildSpec   `json:"spec,omitempty"`
	Status PrebuildStatus `json:"status,omitempty"`
}

type PrebuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Prebuild `json:"items"`
}
