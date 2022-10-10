package business

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	osapps_v1 "github.com/openshift/api/apps/v1"
	osproject_v1 "github.com/openshift/api/project/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	apps_v1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	core_v1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/kubernetes/kubetest"
	"github.com/kiali/kiali/models"
	"github.com/kiali/kiali/prometheus/prometheustest"
)

func setupWorkloadService(k8s *kubetest.K8SClientMock) WorkloadService {
	prom := new(prometheustest.PromClientMock)
	return WorkloadService{k8s: k8s, prom: prom, businessLayer: NewWithBackends(k8s, prom, nil)}
}

func callStreamPodLogs(svc WorkloadService, namespace, podName string, opts *LogOptions) PodLog {
	w := httptest.NewRecorder()
	_ = svc.StreamPodLogs(namespace, podName, opts, w)

	response := w.Result()
	body, _ := io.ReadAll(response.Body)

	var podLogs PodLog
	_ = json.Unmarshal(body, &podLogs)

	return podLogs
}

func TestGetWorkloadListFromDeployments(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeDeployments(), nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.ReplicaSet{}, nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.Pod{}, nil)
	k8s.On("GetPod", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(core_v1.Pod{}, nil)
	k8s.On("GetPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(&kubernetes.PodLogs{}, nil)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	assert.Equal("Namespace", workloadList.Namespace.Name)

	assert.Equal(3, len(workloads))
	assert.Equal("httpbin-v1", workloads[0].Name)
	assert.Equal(true, workloads[0].AppLabel)
	assert.Equal(false, workloads[0].VersionLabel)
	assert.Equal("Deployment", workloads[0].Type)
	assert.Equal("httpbin-v2", workloads[1].Name)
	assert.Equal(true, workloads[1].AppLabel)
	assert.Equal(true, workloads[1].VersionLabel)
	assert.Equal("Deployment", workloads[1].Type)
	assert.Equal("httpbin-v3", workloads[2].Name)
	assert.Equal(false, workloads[2].AppLabel)
	assert.Equal(false, workloads[2].VersionLabel)
	assert.Equal("Deployment", workloads[2].Type)
}

func TestGetWorkloadListFromReplicaSets(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeReplicaSets(), nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.Pod{}, nil)
	k8s.On("GetPod", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(core_v1.Pod{}, nil)
	k8s.On("GetPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(&kubernetes.PodLogs{}, nil)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	assert.Equal("Namespace", workloadList.Namespace.Name)

	assert.Equal(3, len(workloads))
	assert.Equal("httpbin-v1", workloads[0].Name)
	assert.Equal(true, workloads[0].AppLabel)
	assert.Equal(false, workloads[0].VersionLabel)
	assert.Equal("ReplicaSet", workloads[0].Type)
	assert.Equal("httpbin-v2", workloads[1].Name)
	assert.Equal(true, workloads[1].AppLabel)
	assert.Equal(true, workloads[1].VersionLabel)
	assert.Equal("ReplicaSet", workloads[1].Type)
	assert.Equal("httpbin-v3", workloads[2].Name)
	assert.Equal(false, workloads[2].AppLabel)
	assert.Equal(false, workloads[2].VersionLabel)
	assert.Equal("ReplicaSet", workloads[2].Type)
}

func TestGetWorkloadListFromReplicationControllers(t *testing.T) {
	assert := assert.New(t)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.ReplicaSet{}, nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeReplicationControllers(), nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.Pod{}, nil)

	svc := setupWorkloadService(k8s)

	excludedWorkloads = map[string]bool{}
	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	assert.Equal("Namespace", workloadList.Namespace.Name)

	assert.Equal(3, len(workloads))
	assert.Equal("httpbin-v1", workloads[0].Name)
	assert.Equal(true, workloads[0].AppLabel)
	assert.Equal(false, workloads[0].VersionLabel)
	assert.Equal("ReplicationController", workloads[0].Type)
	assert.Equal("httpbin-v2", workloads[1].Name)
	assert.Equal(true, workloads[1].AppLabel)
	assert.Equal(true, workloads[1].VersionLabel)
	assert.Equal("ReplicationController", workloads[1].Type)
	assert.Equal("httpbin-v3", workloads[2].Name)
	assert.Equal(false, workloads[2].AppLabel)
	assert.Equal(false, workloads[2].VersionLabel)
	assert.Equal("ReplicationController", workloads[2].Type)
}

func TestGetWorkloadListFromDeploymentConfigs(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeDeploymentConfigs(), nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.ReplicaSet{}, nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.Pod{}, nil)

	svc := setupWorkloadService(k8s)

	excludedWorkloads = map[string]bool{}
	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	assert.Equal("Namespace", workloadList.Namespace.Name)

	assert.Equal(3, len(workloads))
	assert.Equal("httpbin-v1", workloads[0].Name)
	assert.Equal(true, workloads[0].AppLabel)
	assert.Equal(false, workloads[0].VersionLabel)
	assert.Equal("DeploymentConfig", workloads[0].Type)
	assert.Equal("httpbin-v2", workloads[1].Name)
	assert.Equal(true, workloads[1].AppLabel)
	assert.Equal(true, workloads[1].VersionLabel)
	assert.Equal("DeploymentConfig", workloads[1].Type)
	assert.Equal("httpbin-v3", workloads[2].Name)
	assert.Equal(false, workloads[2].AppLabel)
	assert.Equal(false, workloads[2].VersionLabel)
	assert.Equal("DeploymentConfig", workloads[2].Type)
}

func TestGetWorkloadListFromStatefulSets(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.ReplicaSet{}, nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeStatefulSets(), nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.Pod{}, nil)

	svc := setupWorkloadService(k8s)

	excludedWorkloads = map[string]bool{}
	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	assert.Equal("Namespace", workloadList.Namespace.Name)

	assert.Equal(3, len(workloads))
	assert.Equal("httpbin-v1", workloads[0].Name)
	assert.Equal(true, workloads[0].AppLabel)
	assert.Equal(false, workloads[0].VersionLabel)
	assert.Equal("StatefulSet", workloads[0].Type)
	assert.Equal("httpbin-v2", workloads[1].Name)
	assert.Equal(true, workloads[1].AppLabel)
	assert.Equal(true, workloads[1].VersionLabel)
	assert.Equal("StatefulSet", workloads[1].Type)
	assert.Equal("httpbin-v3", workloads[2].Name)
	assert.Equal(false, workloads[2].AppLabel)
	assert.Equal(false, workloads[2].VersionLabel)
	assert.Equal("StatefulSet", workloads[2].Type)
}

func TestGetWorkloadListFromDaemonSets(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.ReplicaSet{}, nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeDaemonSets(), nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.Pod{}, nil)

	svc := setupWorkloadService(k8s)

	excludedWorkloads = map[string]bool{}
	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	assert.Equal("Namespace", workloadList.Namespace.Name)

	assert.Equal(3, len(workloads))
	assert.Equal("httpbin-v1", workloads[0].Name)
	assert.Equal(true, workloads[0].AppLabel)
	assert.Equal(false, workloads[0].VersionLabel)
	assert.Equal("DaemonSet", workloads[0].Type)
	assert.Equal("httpbin-v2", workloads[1].Name)
	assert.Equal(true, workloads[1].AppLabel)
	assert.Equal(true, workloads[1].VersionLabel)
	assert.Equal("DaemonSet", workloads[1].Type)
	assert.Equal("httpbin-v3", workloads[2].Name)
	assert.Equal(false, workloads[2].AppLabel)
	assert.Equal(false, workloads[2].VersionLabel)
	assert.Equal("DaemonSet", workloads[2].Type)
}

func TestGetWorkloadListFromDepRCPod(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeDepSyncedWithRS(), nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeRSSyncedWithPods(), nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakePodsSyncedWithDeployments(), nil)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	assert.Equal("Namespace", workloadList.Namespace.Name)

	assert.Equal(1, len(workloads))
	assert.Equal("details-v1", workloads[0].Name)
	assert.Equal("Deployment", workloads[0].Type)
	assert.Equal(true, workloads[0].AppLabel)
	assert.Equal(true, workloads[0].VersionLabel)
}

func TestGetWorkloadListFromPod(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.ReplicaSet{}, nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakePodsNoController(), nil)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	assert.Equal("Namespace", workloadList.Namespace.Name)

	assert.Equal(1, len(workloads))
	assert.Equal("orphan-pod", workloads[0].Name)
	assert.Equal("Pod", workloads[0].Type)
	assert.Equal(true, workloads[0].AppLabel)
	assert.Equal(true, workloads[0].VersionLabel)
}

func TestGetWorkloadListFromPods(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeCustomControllerRSSyncedWithPods(), nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakePodsFromCustomController(), nil)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	assert.Equal("Namespace", workloadList.Namespace.Name)

	assert.Equal(1, len(workloads))
	assert.Equal("custom-controller-RS-123", workloads[0].Name)
	assert.Equal("ReplicaSet", workloads[0].Type)
	assert.Equal(true, workloads[0].AppLabel)
	assert.Equal(true, workloads[0].VersionLabel)
}

func TestGetWorkloadFromDeployment(t *testing.T) {
	assert := assert.New(t)

	// Setup mocks
	gr := schema.GroupResource{
		Group:    "test-group",
		Resource: "test-resource",
	}
	notfound := errors.NewNotFound(gr, "not found")
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{ObjectMeta: v1.ObjectMeta{Name: "Namespace"}}, nil)
	k8s.On("GetDeployment", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&FakeDepSyncedWithRS()[0], nil)
	k8s.On("GetDeploymentConfig", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&osapps_v1.DeploymentConfig{}, notfound)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeRSSyncedWithPods(), nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.StatefulSet{}, notfound)
	k8s.On("GetDaemonSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.DaemonSet{}, notfound)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakePodsSyncedWithDeployments(), nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)

	// Disabling CustomDashboards on Workload details testing
	conf := config.Get()
	conf.ExternalServices.CustomDashboards.Enabled = false
	config.Set(conf)

	svc := setupWorkloadService(k8s)
	criteria := WorkloadCriteria{Namespace: "Namespace", WorkloadName: "details-v1", WorkloadType: "", IncludeServices: false}
	workload, _ := svc.GetWorkload(context.TODO(), criteria)

	assert.Equal("details-v1", workload.Name)
	assert.Equal("Deployment", workload.Type)
	assert.Equal(true, workload.AppLabel)
	assert.Equal(true, workload.VersionLabel)
}

func TestGetWorkloadWithInvalidWorkloadType(t *testing.T) {
	assert := assert.New(t)

	// Setup mocks
	gr := schema.GroupResource{
		Group:    "test-group",
		Resource: "test-resource",
	}
	notfound := errors.NewNotFound(gr, "not found")
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployment", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&FakeDepSyncedWithRS()[0], nil)
	k8s.On("GetDeploymentConfig", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&osapps_v1.DeploymentConfig{}, notfound)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeRSSyncedWithPods(), nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.StatefulSet{}, notfound)
	k8s.On("GetDaemonSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.DaemonSet{}, notfound)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakePodsSyncedWithDeployments(), nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)

	// Disabling CustomDashboards on Workload details testing
	conf := config.Get()
	conf.ExternalServices.CustomDashboards.Enabled = false
	config.Set(conf)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", WorkloadName: "details-v1", WorkloadType: "invalid", IncludeServices: false}
	workload, _ := svc.GetWorkload(context.TODO(), criteria)

	assert.Equal("details-v1", workload.Name)
	assert.Equal("Deployment", workload.Type)
	assert.Equal(true, workload.AppLabel)
	assert.Equal(true, workload.VersionLabel)
}

func TestGetWorkloadFromPods(t *testing.T) {
	assert := assert.New(t)

	// Setup mocks
	gr := schema.GroupResource{
		Group:    "test-group",
		Resource: "test-resource",
	}
	notfound := errors.NewNotFound(gr, "not found")
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployment", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.Deployment{}, notfound)
	k8s.On("GetDeploymentConfig", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&osapps_v1.DeploymentConfig{}, notfound)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeCustomControllerRSSyncedWithPods(), nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.StatefulSet{}, notfound)
	k8s.On("GetDaemonSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.DaemonSet{}, notfound)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakePodsFromCustomController(), nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)

	// Disabling CustomDashboards on Workload details testing
	conf := config.Get()
	conf.ExternalServices.CustomDashboards.Enabled = false
	config.Set(conf)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", WorkloadName: "custom-controller", WorkloadType: "", IncludeServices: false}
	workload, _ := svc.GetWorkload(context.TODO(), criteria)

	// custom controller is not a workload type, only its replica set(s)
	assert.Equal((*models.Workload)(nil), workload)

	criteria = WorkloadCriteria{Namespace: "Namespace", WorkloadName: "custom-controller-RS-123", WorkloadType: "", IncludeServices: false}
	workload, _ = svc.GetWorkload(context.TODO(), criteria)

	assert.Equal("custom-controller-RS-123", workload.Name)
	assert.Equal("ReplicaSet", workload.Type)
	assert.Equal(true, workload.AppLabel)
	assert.Equal(true, workload.VersionLabel)
	assert.Equal(0, len(workload.Runtimes))
	assert.Equal(0, len(workload.AdditionalDetails))
}

func TestGetPods(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakePodsSyncedWithDeployments(), nil)
	k8s.On("IsOpenShift").Return(false)
	k8s.On("IsGatewayAPI").Return(false)

	svc := setupWorkloadService(k8s)

	pods, _ := svc.GetPods(context.TODO(), "Namespace", "app=httpbin")

	assert.Equal(1, len(pods))
	assert.Equal("details-v1-3618568057-dnkjp", pods[0].Name)
}

func TestGetPod(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("GetPod", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakePodSyncedWithDeployments(), nil)
	k8s.On("IsOpenShift").Return(false)
	k8s.On("IsGatewayAPI").Return(false)

	svc := setupWorkloadService(k8s)

	pod, _ := svc.GetPod("Namespace", "details-v1-3618568057-dnkjp")

	assert.Equal("details-v1-3618568057-dnkjp", pod.Name)
}

func TestGetPodLogs(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	fplswd := FakePodLogsSyncedWithDeployments()
	k8s.On("StreamPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(io.NopCloser(strings.NewReader(fplswd.Logs)), nil)
	k8s.On("IsOpenShift").Return(false)
	k8s.On("IsGatewayAPI").Return(false)

	svc := setupWorkloadService(k8s)
	podLogs := callStreamPodLogs(svc, "Namespace", "details-v1-3618568057-dnkjp", &LogOptions{PodLogOptions: core_v1.PodLogOptions{Container: "details"}})

	assert.Equal(len(podLogs.Entries), 4)

	assert.Equal("2018-01-02 03:34:28.000", podLogs.Entries[0].Timestamp)
	assert.Equal(int64(1514864068000), podLogs.Entries[0].TimestampUnix)
	assert.Equal("INFO #1 Log Message", podLogs.Entries[0].Message)
	assert.Equal("INFO", podLogs.Entries[0].Severity)

	assert.Equal("2018-01-02 04:34:28.000", podLogs.Entries[1].Timestamp)
	assert.Equal(int64(1514867668000), podLogs.Entries[1].TimestampUnix)
	assert.Equal("WARN #2 Log Message", podLogs.Entries[1].Message)
	assert.Equal("WARN", podLogs.Entries[1].Severity)

	assert.Equal("2018-01-02 05:34:28.000", podLogs.Entries[2].Timestamp)
	assert.Equal(int64(1514871268000), podLogs.Entries[2].TimestampUnix)
	assert.Equal("#3 Log Message", podLogs.Entries[2].Message)
	assert.Equal("INFO", podLogs.Entries[2].Severity)

	assert.Equal("2018-01-02 06:34:28.000", podLogs.Entries[3].Timestamp)
	assert.Equal(int64(1514874868000), podLogs.Entries[3].TimestampUnix)
	assert.Equal("#4 Log error Message", podLogs.Entries[3].Message)
	assert.Equal("ERROR", podLogs.Entries[3].Severity)
}

func TestGetPodLogsMaxLines(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	fplswd := FakePodLogsSyncedWithDeployments()
	k8s.On("StreamPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(io.NopCloser(strings.NewReader(fplswd.Logs)), nil)
	k8s.On("IsOpenShift").Return(false)
	k8s.On("IsGatewayAPI").Return(false)

	svc := setupWorkloadService(k8s)

	maxLines := 2
	duration, _ := time.ParseDuration("6h")
	podLogs := callStreamPodLogs(svc, "Namespace", "details-v1-3618568057-dnkjp", &LogOptions{PodLogOptions: core_v1.PodLogOptions{Container: "details"}, MaxLines: &maxLines, Duration: &duration})

	assert.Equal(2, len(podLogs.Entries))
	assert.Equal("INFO #1 Log Message", podLogs.Entries[0].Message)
	assert.Equal("WARN #2 Log Message", podLogs.Entries[1].Message)
}

func TestGetPodLogsDuration(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	fplswd := FakePodLogsSyncedWithDeployments()
	k8s.On("StreamPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(io.NopCloser(strings.NewReader(fplswd.Logs)), nil)
	k8s.On("IsOpenShift").Return(false)
	k8s.On("IsGatewayAPI").Return(false)
	svc := setupWorkloadService(k8s)

	duration, _ := time.ParseDuration("59m")
	podLogs := callStreamPodLogs(svc, "Namespace", "details-v1-3618568057-dnkjp", &LogOptions{PodLogOptions: core_v1.PodLogOptions{Container: "details"}, Duration: &duration})
	assert.Equal(1, len(podLogs.Entries))
	assert.Equal("INFO #1 Log Message", podLogs.Entries[0].Message)

	// Re-setup mocks
	k8s = new(kubetest.K8SClientMock)
	k8s.On("StreamPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(io.NopCloser(strings.NewReader(fplswd.Logs)), nil)
	k8s.On("IsOpenShift").Return(false)
	k8s.On("IsGatewayAPI").Return(false)
	svc = setupWorkloadService(k8s)

	duration, _ = time.ParseDuration("1h")
	podLogs = callStreamPodLogs(svc, "Namespace", "details-v1-3618568057-dnkjp", &LogOptions{PodLogOptions: core_v1.PodLogOptions{Container: "details"}, Duration: &duration})
	assert.Equal(2, len(podLogs.Entries))
	assert.Equal("INFO #1 Log Message", podLogs.Entries[0].Message)
	assert.Equal("WARN #2 Log Message", podLogs.Entries[1].Message)

	// Re-setup mocks
	k8s = new(kubetest.K8SClientMock)
	k8s.On("StreamPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(io.NopCloser(strings.NewReader(fplswd.Logs)), nil)
	k8s.On("IsOpenShift").Return(false)
	k8s.On("IsGatewayAPI").Return(false)
	svc = setupWorkloadService(k8s)

	duration, _ = time.ParseDuration("2h")
	podLogs = callStreamPodLogs(svc, "Namespace", "details-v1-3618568057-dnkjp", &LogOptions{PodLogOptions: core_v1.PodLogOptions{Container: "details"}, Duration: &duration})
	assert.Equal(3, len(podLogs.Entries))
	assert.Equal("INFO #1 Log Message", podLogs.Entries[0].Message)
	assert.Equal("WARN #2 Log Message", podLogs.Entries[1].Message)
	assert.Equal("#3 Log Message", podLogs.Entries[2].Message)
}

func TestGetPodLogsMaxLinesAndDurations(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	fplswd := FakePodLogsSyncedWithDeployments()
	k8s.On("StreamPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(io.NopCloser(strings.NewReader(fplswd.Logs)), nil)
	k8s.On("IsOpenShift").Return(false)
	k8s.On("IsGatewayAPI").Return(false)
	svc := setupWorkloadService(k8s)

	maxLines := 2
	duration, _ := time.ParseDuration("2h")
	podLogs := callStreamPodLogs(svc, "Namespace", "details-v1-3618568057-dnkjp", &LogOptions{Duration: &duration, PodLogOptions: core_v1.PodLogOptions{Container: "details"}, MaxLines: &maxLines})
	assert.Equal(2, len(podLogs.Entries))
	assert.Equal("INFO #1 Log Message", podLogs.Entries[0].Message)
	assert.Equal("WARN #2 Log Message", podLogs.Entries[1].Message)
	assert.True(podLogs.LinesTruncated)

	// Re-setup mocks
	k8s = new(kubetest.K8SClientMock)
	k8s.On("StreamPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(io.NopCloser(strings.NewReader(fplswd.Logs)), nil)
	k8s.On("IsOpenShift").Return(false)
	k8s.On("IsGatewayAPI").Return(false)
	svc = setupWorkloadService(k8s)

	maxLines = 3
	duration, _ = time.ParseDuration("3h")
	podLogs = callStreamPodLogs(svc, "Namespace", "details-v1-3618568057-dnkjp", &LogOptions{Duration: &duration, PodLogOptions: core_v1.PodLogOptions{Container: "details"}, MaxLines: &maxLines})
	assert.Equal(3, len(podLogs.Entries))
	assert.Equal("INFO #1 Log Message", podLogs.Entries[0].Message)
	assert.Equal("WARN #2 Log Message", podLogs.Entries[1].Message)
	assert.Equal("#3 Log Message", podLogs.Entries[2].Message)
	assert.False(podLogs.LinesTruncated)
}

func TestGetPodLogsProxy(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	fplp := FakePodLogsProxy()
	k8s.On("StreamPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(io.NopCloser(strings.NewReader(fplp.Logs)), nil)
	k8s.On("IsOpenShift").Return(false)
	k8s.On("IsGatewayAPI").Return(false)

	svc := setupWorkloadService(k8s)

	maxLines := 2
	duration, _ := time.ParseDuration("2h")
	podLogs := callStreamPodLogs(svc, "Namespace", "details-v1-3618568057-dnkjp", &LogOptions{Duration: &duration, IsProxy: true, PodLogOptions: core_v1.PodLogOptions{Container: "details"}, MaxLines: &maxLines})
	assert.Equal(1, len(podLogs.Entries))
	entry := podLogs.Entries[0]
	assert.Equal(`[2021-02-01T21:34:35.533Z] "GET /hotels/Ljubljana HTTP/1.1" 200 - via_upstream - "-" 0 99 14 14 "-" "Go-http-client/1.1" "7e7e2dd0-0a96-4535-950b-e303805b7e27" "hotels.travel-agency:8000" "127.0.2021-02-01T21:34:38.761055140Z 0.1:8000" inbound|8000|| 127.0.0.1:33704 10.129.0.72:8000 10.128.0.79:39880 outbound_.8000_._.hotels.travel-agency.svc.cluster.local default`, entry.Message)
	assert.Equal("2021-02-01 21:34:35.533", entry.Timestamp)
	assert.NotNil(entry.AccessLog)
	assert.Equal("GET", entry.AccessLog.Method)
	assert.Equal("200", entry.AccessLog.StatusCode)
	assert.Equal("2021-02-01T21:34:35.533Z", entry.AccessLog.Timestamp)
	assert.Equal(int64(1612215275533), entry.TimestampUnix)
}

func TestDuplicatedControllers(t *testing.T) {
	assert := assert.New(t)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeDuplicatedDeployments(), nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeDuplicatedReplicaSets(), nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakeDuplicatedStatefulSets(), nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakePodsSyncedWithDuplicated(), nil)
	k8s.On("GetPod", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(FakePodSyncedWithDeployments(), nil)
	k8s.On("GetPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(FakePodLogsSyncedWithDeployments(), nil)

	notfound := fmt.Errorf("not found")
	k8s.On("GetDeployment", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&FakeDuplicatedDeployments()[0], nil)
	k8s.On("GetDeploymentConfig", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&osapps_v1.DeploymentConfig{}, notfound)
	k8s.On("GetStatefulSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&FakeDuplicatedStatefulSets()[0], nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetDaemonSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.DaemonSet{}, notfound)

	// Disabling CustomDashboards on Workload details testing
	conf := config.Get()
	conf.ExternalServices.CustomDashboards.Enabled = false
	config.Set(conf)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	criteria = WorkloadCriteria{Namespace: "Namespace", WorkloadName: "duplicated-v1", WorkloadType: "", IncludeServices: false}
	workload, _ := svc.GetWorkload(context.TODO(), criteria)

	assert.Equal(workloads[0].Type, workload.Type)
}

func TestGetWorkloadListFromGenericPodController(t *testing.T) {
	assert := assert.New(t)

	pods := FakePodsSyncedWithDeployments()

	// Doesn't matter what the type is as long as kiali doesn't recognize it as a workload.
	owner := &core_v1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name: "testing",
			UID:  types.UID("f9952f02-5552-4b2c-afdb-441d859dbb36"),
		},
	}
	ref := v1.NewControllerRef(owner, core_v1.SchemeGroupVersion.WithKind("ConfigMap"))

	for i := range pods {
		pods[i].OwnerReferences = []v1.OwnerReference{*ref}
	}

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.ReplicaSet{}, nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(pods, nil)
	k8s.On("GetPod", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(pods[0], nil)
	k8s.On("GetPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(pods, nil)

	notfound := fmt.Errorf("not found")
	k8s.On("GetDeployment", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfig", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&osapps_v1.DeploymentConfig{}, notfound)
	k8s.On("GetStatefulSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetDaemonSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.DaemonSet{}, notfound)

	// Disabling CustomDashboards on Workload details testing
	conf := config.Get()
	conf.ExternalServices.CustomDashboards.Enabled = false
	config.Set(conf)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	criteria = WorkloadCriteria{Namespace: "Namespace", WorkloadName: owner.Name, WorkloadType: "", IncludeServices: false}
	workload, _ := svc.GetWorkload(context.TODO(), criteria)

	assert.Equal(len(workloads), 1)
	assert.NotNil(workload)

	assert.Equal(len(pods), len(workload.Pods))
}

func TestGetWorkloadListKindsWithSameName(t *testing.T) {
	assert := assert.New(t)

	rs := FakeRSSyncedWithPods()
	pods := FakePodsSyncedWithDeployments()
	pods[0].OwnerReferences[0].APIVersion = "shiny.new.apps/v1"
	pods[0].OwnerReferences[0].Kind = "ReplicaSet"

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(rs, nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(pods, nil)
	k8s.On("GetPod", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(pods[0], nil)
	k8s.On("GetPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(pods, nil)

	notfound := fmt.Errorf("not found")
	k8s.On("GetDeployment", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfig", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&osapps_v1.DeploymentConfig{}, notfound)
	k8s.On("GetStatefulSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetDaemonSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.DaemonSet{}, notfound)

	// Disabling CustomDashboards on Workload details testing
	conf := config.Get()
	conf.ExternalServices.CustomDashboards.Enabled = false
	config.Set(conf)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	assert.Equal(0, len(workloads))
}

func TestGetWorkloadListRSWithoutPrefix(t *testing.T) {
	assert := assert.New(t)

	rs := FakeRSSyncedWithPods()
	// Doesn't matter what the type is as long as kiali doesn't recognize it as a workload.
	owner := &core_v1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			// Random prefix
			Name: "h79a3h-controlling-workload",
			UID:  types.UID("f9952f02-5552-4b2c-afdb-441d859dbb36"),
		},
		TypeMeta: v1.TypeMeta{
			Kind: "ConfigMap",
		},
	}
	rs[0].OwnerReferences = []v1.OwnerReference{*v1.NewControllerRef(owner, core_v1.SchemeGroupVersion.WithKind(owner.Kind))}
	pods := FakePodsSyncedWithDeployments()

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(rs, nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(pods, nil)
	k8s.On("GetPod", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(pods[0], nil)
	k8s.On("GetPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(pods, nil)

	notfound := fmt.Errorf("not found")
	k8s.On("GetDeployment", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfig", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&osapps_v1.DeploymentConfig{}, notfound)
	k8s.On("GetStatefulSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetDaemonSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.DaemonSet{}, notfound)

	// Disabling CustomDashboards on Workload details testing
	conf := config.Get()
	conf.ExternalServices.CustomDashboards.Enabled = false
	config.Set(conf)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	assert.Equal(1, len(workloads))
}

func TestGetWorkloadListRSOwnedByCustom(t *testing.T) {
	assert := assert.New(t)

	replicaSets := FakeRSSyncedWithPods()

	// Doesn't matter what the type is as long as kiali doesn't recognize it as a workload.
	owner := &core_v1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name: "controlling-workload",
			UID:  types.UID("f9952f02-5552-4b2c-afdb-441d859dbb36"),
		},
		TypeMeta: v1.TypeMeta{
			Kind: "ConfigMap",
		},
	}
	ref := v1.NewControllerRef(owner, core_v1.SchemeGroupVersion.WithKind(owner.Kind))

	for i := range replicaSets {
		replicaSets[i].OwnerReferences = []v1.OwnerReference{*ref}
	}

	pods := FakePodsSyncedWithDeployments()

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	k8s.On("IsOpenShift").Return(true)
	k8s.On("IsGatewayAPI").Return(false)
	k8s.On("GetProject", mock.AnythingOfType("string")).Return(&osproject_v1.Project{}, nil)
	k8s.On("GetDeployments", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfigs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]osapps_v1.DeploymentConfig{}, nil)
	k8s.On("GetReplicaSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(replicaSets, nil)
	k8s.On("GetReplicationControllers", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]core_v1.ReplicationController{}, nil)
	k8s.On("GetStatefulSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.StatefulSet{}, nil)
	k8s.On("GetJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.Job{}, nil)
	k8s.On("GetCronJobs", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]batch_v1.CronJob{}, nil)
	k8s.On("GetPods", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(pods, nil)
	k8s.On("GetPod", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(pods[0], nil)
	k8s.On("GetPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(FakePodsSyncedWithDeployments(), nil)

	notfound := fmt.Errorf("not found")
	k8s.On("GetDeployment", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.Deployment{}, nil)
	k8s.On("GetDeploymentConfig", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&osapps_v1.DeploymentConfig{}, notfound)
	k8s.On("GetStatefulSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.StatefulSet{}, nil)
	k8s.On("GetDaemonSets", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]apps_v1.DaemonSet{}, nil)
	k8s.On("GetDaemonSet", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&apps_v1.DaemonSet{}, notfound)

	// Disabling CustomDashboards on Workload details testing
	conf := config.Get()
	conf.ExternalServices.CustomDashboards.Enabled = false
	config.Set(conf)

	svc := setupWorkloadService(k8s)

	criteria := WorkloadCriteria{Namespace: "Namespace", IncludeIstioResources: false, IncludeHealth: false}
	workloadList, _ := svc.GetWorkloadList(context.TODO(), criteria)
	workloads := workloadList.Workloads

	criteria = WorkloadCriteria{Namespace: "Namespace", WorkloadName: owner.Name, WorkloadType: "", IncludeServices: false}
	workload, _ := svc.GetWorkload(context.TODO(), criteria)

	assert.Equal(len(workloads), 1)
	assert.Nil(workload)

	criteria.WorkloadName = workloads[0].Name
	workload, _ = svc.GetWorkload(context.TODO(), criteria)

	assert.NotNil(workload)
}

func TestGetPodLogsWithoutAccessLogs(t *testing.T) {
	assert := assert.New(t)
	conf := config.NewConfig()
	config.Set(conf)

	// Setup mocks
	k8s := new(kubetest.K8SClientMock)
	const logs = `2021-10-05T00:32:40.309334Z     debug   envoy http      [C57][S7648448766062793478] request end stream
2021-10-05T00:32:40.309425Z     debug   envoy router    [C57][S7648448766062793478] cluster 'inbound|9080||' match for URL '/details/0'
2021-10-05T00:32:40.309438Z     debug   envoy upstream  Using existing host 172.17.0.12:9080.
2021-10-05T00:32:40.309457Z     debug   envoy router    [C57][S7648448766062793478] router decoding headers:
2021-10-05T00:32:40.309457Z     ':authority', 'details:9080'
2021-10-05T00:32:40.309457Z     ':path', '/details/0'
2021-10-05T00:32:40.309457Z     ':method', 'GET'
2021-10-05T00:32:40.309457Z     ':scheme', 'http'`
	k8s.On("StreamPodLogs", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(io.NopCloser(strings.NewReader(logs)), nil)
	k8s.On("IsOpenShift").Return(false)
	k8s.On("IsGatewayAPI").Return(false)

	svc := setupWorkloadService(k8s)

	podLogs := callStreamPodLogs(svc, "Namespace", "details-v1-3618568057-dnkjp", &LogOptions{IsProxy: true, PodLogOptions: core_v1.PodLogOptions{Container: "istio-proxy"}})

	assert.Equal(8, len(podLogs.Entries))
	for _, entry := range podLogs.Entries {
		assert.Nil(entry.AccessLog)
	}
}

func TestFilterUniqueIstioReferences(t *testing.T) {
	assert := assert.New(t)
	references := []*models.IstioValidationKey{
		{ObjectType: "t1", Namespace: "ns1", Name: "n1"},
		{ObjectType: "t1", Namespace: "ns1", Name: "n1"},
		{ObjectType: "t2", Namespace: "ns2", Name: "n2"},
	}
	filtered := FilterUniqueIstioReferences(references)
	assert.Equal(2, len(filtered))
}
