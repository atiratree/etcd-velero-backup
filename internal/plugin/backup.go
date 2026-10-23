package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type BackupPlugin struct {
	log logrus.FieldLogger
}

func NewBackupPlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &BackupPlugin{log: logger}, nil
}

func (p *BackupPlugin) Name() string {
	return "etcd-backup"
}

func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources:  []string{"pods"},
		LabelSelector:      "etcd=true,app=etcd",
		IncludedNamespaces: []string{"openshift-etcd"},
	}, nil
}

func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *velerov1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	p.log.Info("Executing etcd backup plugin")

	pod := corev1API.Pod{}
	itemMarshal, _ := json.Marshal(item)
	err := json.Unmarshal(itemMarshal, &pod)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get unmarshal pod: %w", err)
	}

	client, err := GetKubeClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	backupName := backup.Name // TODO: use in the backup filename?
	jobName := SanitizeName(backupName, "etcd-backup-")

	err = CreateJob(
		context.TODO(),
		client,
		p.log,
		pod.Namespace,
		jobName,
		pod.Spec.NodeName,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create backup job: %w", err)
	}

	p.log.Infof("Waiting for backup job %s/%s to complete", pod.Namespace, jobName)
	if err := waitForJob(context.TODO(), client, p.log, pod.Namespace, jobName); err != nil {
		return nil, nil, err
	}

	return item, nil, nil
}
