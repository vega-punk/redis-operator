package k8s

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"redis-operator/log"
	"redis-operator/metrics"
)

// StatefulSet the StatefulSet service that knows how to interact with k8s to manage them
type StatefulSet interface {
	GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error)
	GetStatefulSetPods(namespace, name string) (*corev1.PodList, error)
	CreateStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error
	UpdateStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error
	CreateOrUpdateStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error
	DeleteStatefulSet(namespace string, name string) error
	ListStatefulSets(namespace string) (*appsv1.StatefulSetList, error)
}

// StatefulSetService is the service account service implementation using API calls to kubernetes.
type StatefulSetService struct {
	kubeClient      kubernetes.Interface
	logger          log.Logger
	metricsRecorder metrics.Recorder
}

// NewStatefulSetService returns a new StatefulSet KubeService.
func NewStatefulSetService(kubeClient kubernetes.Interface, logger log.Logger, metricsRecorder metrics.Recorder) *StatefulSetService {
	logger = logger.With("service", "k8s.statefulSet")
	return &StatefulSetService{
		kubeClient:      kubeClient,
		logger:          logger,
		metricsRecorder: metricsRecorder,
	}
}

// GetStatefulSet will retrieve the requested statefulset based on namespace and name
func (s *StatefulSetService) GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	statefulSet, err := s.kubeClient.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	recordMetrics(namespace, "StatefulSet", name, "GET", err, s.metricsRecorder)
	if err != nil {
		return nil, err
	}
	return statefulSet, err
}

// GetStatefulSetPods will give a list of pods that are managed by the statefulset
func (s *StatefulSetService) GetStatefulSetPods(namespace, name string) (*corev1.PodList, error) {
	statefulSet, err := s.GetStatefulSet(namespace, name)
	if err != nil {
		return nil, err
	}
	labels := []string{}
	for k, v := range statefulSet.Spec.Selector.MatchLabels {
		labels = append(labels, fmt.Sprintf("%s=%s", k, v))
	}
	selector := strings.Join(labels, ",")
	return s.kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
}

// CreateStatefulSet will create the given statefulset
func (s *StatefulSetService) CreateStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error {
	_, err := s.kubeClient.AppsV1().StatefulSets(namespace).Create(context.TODO(), statefulSet, metav1.CreateOptions{})
	recordMetrics(namespace, "StatefulSet", statefulSet.GetName(), "CREATE", err, s.metricsRecorder)
	if err != nil {
		return err
	}
	s.logger.WithField("namespace", namespace).WithField("statefulSet", statefulSet.ObjectMeta.Name).Infof("statefulSet created")
	return err
}

// UpdateStatefulSet will update the given statefulset
func (s *StatefulSetService) UpdateStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error {
	_, err := s.kubeClient.AppsV1().StatefulSets(namespace).Update(context.TODO(), statefulSet, metav1.UpdateOptions{})
	recordMetrics(namespace, "StatefulSet", statefulSet.GetName(), "UPDATE", err, s.metricsRecorder)
	if err != nil {
		return err
	}
	s.logger.WithField("namespace", namespace).WithField("statefulSet", statefulSet.ObjectMeta.Name).Infof("statefulSet updated")
	return err
}

// CreateOrUpdateStatefulSet will update the statefulset or create it if does not exist
func (s *StatefulSetService) CreateOrUpdateStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error {
	storedStatefulSet, err := s.GetStatefulSet(namespace, statefulSet.Name)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return s.CreateStatefulSet(namespace, statefulSet)
		}
		return err
	}

	// Already exists, need to Update.
	// Set the correct resource version to ensure we are on the latest version. This way the only valid
	// namespace is our spec(https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency),
	// we will replace the current namespace state.
	statefulSet.ResourceVersion = storedStatefulSet.ResourceVersion
	return s.UpdateStatefulSet(namespace, statefulSet)
}

// DeleteStatefulSet will delete the statefulset
func (s *StatefulSetService) DeleteStatefulSet(namespace, name string) error {
	propagation := metav1.DeletePropagationForeground
	err := s.kubeClient.AppsV1().StatefulSets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{PropagationPolicy: &propagation})
	recordMetrics(namespace, "StatefulSet", name, "DELETE", err, s.metricsRecorder)
	return err
}

// ListStatefulSets will retrieve a list of statefulset in the given namespace
func (s *StatefulSetService) ListStatefulSets(namespace string) (*appsv1.StatefulSetList, error) {
	stsList, err := s.kubeClient.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	recordMetrics(namespace, "StatefulSet", metrics.NOT_APPLICABLE, "LIST", err, s.metricsRecorder)
	return stsList, err
}
