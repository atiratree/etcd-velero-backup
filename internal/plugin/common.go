package plugin

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	ConfigLabel = "etcd-backup.io/component"

	BackupMountPath = "/tmp/backup"
	jobPollInterval = 5 * time.Second
	jobTimeout      = 30 * time.Minute
)

var (
	kubeClient kubernetes.Interface
	kubeOnce   sync.Once
	kubeErr    error
)

func GetKubeClient() (kubernetes.Interface, error) {
	kubeOnce.Do(func() {
		config, err := rest.InClusterConfig()
		if err != nil {
			kubeErr = fmt.Errorf("failed to get in-cluster config: %w", err)
			return
		}
		kubeClient, kubeErr = kubernetes.NewForConfig(config)
	})
	return kubeClient, kubeErr
}

func CreateJob(ctx context.Context, client kubernetes.Interface, log logrus.FieldLogger, namespace, jobName, nodeName string) error {
	job := buildJob(jobName, namespace, nodeName)

	_, err := client.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create job %s/%s: %w", namespace, jobName, err)
	}

	log.Infof("Created job %s/%s", namespace, jobName)
	return nil
}

func buildJob(name, namespace, nodename string) *batchv1.Job {
	privileged := true
	runAsRoot := int64(0)
	backoffLimit := int32(0)

	hostPathType := corev1.HostPathDirectoryOrCreate

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ConfigLabel: "job",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					HostNetwork:   true,
					HostPID:       true,
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": nodename,
					},
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master",
							Effect: "NoSchedule",
						},
					},
					Containers: []corev1.Container{
						{
							Name:    "etcd-backup",
							Image:   "openshift4/ose-cli-artifacts-rhel9",
							Command: []string{"/bin/bash"},
							Args: []string{"-c", `
/usr/local/bin//cluster-backup.sh ${BACKUP_DIR}
`},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
								RunAsUser:  &runAsRoot,
							},
							Env: []corev1.EnvVar{
								{Name: "BACKUP_DIR", Value: BackupMountPath},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "usr-local-bin",
									MountPath: "/usr/local/bin",
								},
								{
									Name:      "resources-dir",
									MountPath: "/etc/kubernetes/static-pod-resources",
								},
								{
									Name:      "cert-dir",
									MountPath: "/etc/kubernetes/static-pod-certs",
								},
								{
									Name:      "static-pod-dir",
									MountPath: "/etc/kubernetes/manifests",
								},
								{
									Name:      "backup-storage",
									MountPath: BackupMountPath,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "backup-storage",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: BackupMountPath,
									Type: &hostPathType,
								},
							},
						},
						{
							Name: "usr-local-bin",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/usr/local/bin",
								},
							},
						},
						{
							Name: "static-pod-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/kubernetes/manifests",
								},
							},
						},
						{
							Name: "resources-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/kubernetes/static-pod-resources",
								},
							},
						},
						{
							Name: "cert-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/kubernetes/static-pod-resources/etcd-certs",
								},
							},
						},
					},
				},
			},
		},
	}
}

func waitForJob(ctx context.Context, client kubernetes.Interface, log logrus.FieldLogger, namespace, jobName string) error {
	timeout := time.After(jobTimeout)
	ticker := time.NewTicker(jobPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for job %s/%s after %s", namespace, jobName, jobTimeout)
		case <-ticker.C:
			completed, failed, message, err := GetJobStatus(ctx, client, namespace, jobName)
			if err != nil {
				return err
			}
			if completed && failed {
				return fmt.Errorf("job %s/%s failed: %s", namespace, jobName, message)
			}
			if completed {
				return nil
			}
			log.Infof("Job %s/%s still running...", namespace, jobName)
		}
	}
}

func GetJobStatus(ctx context.Context, client kubernetes.Interface, namespace, jobName string) (completed bool, failed bool, message string, err error) {
	job, err := client.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return false, false, "", fmt.Errorf("failed to get job %s/%s: %w", namespace, jobName, err)
	}

	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return true, false, "job completed successfully", nil
		}
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return true, true, condition.Message, nil
		}
	}

	return false, false, "job still running", nil
}

func SanitizeName(name string, prefix string) string {
	result := prefix + strings.ToLower(name)
	result = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, result)
	if len(result) > 63 {
		result = result[:63]
	}
	result = strings.TrimRight(result, "-")
	return result
}
