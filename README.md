# Experimental OpenShift etcd Backup with OADP/Velero

PoC assisted by Claude.

### Setup

Set a REGISTRY in Makefile

```bash
make push
```

Add a custom plugin to your DataProtectionApplication
- https://docs.redhat.com/en/documentation/openshift_container_platform/4.22/html/backup_and_restore/oadp-application-backup-and-restore#custom-velero-plugins

```yaml
kind: DataProtectionApplication
apiVersion: oadp.openshift.io/v1alpha1
metadata:
  name: velero-sample
  namespace: openshift-adp
spec:
  configuration:
    velero:
      customPlugins:
        - name: etcd-backup-plugin
          image: quay.io/<name>/etcd-backup-plugin:latest
      noDefaultBackupLocation: true
```

Ensure BackupStorageLocation exists

```bash
kubectl apply -f - <<EOF
kind: BackupStorageLocation
apiVersion: velero.io/v1
metadata:
  name: backupstoragelocation
  namespace: openshift-adp
spec: 
  default: true
  provider: etcd-backup.io/local-store
  objectStorage:
    bucket: etcd-backups
EOF
```


### Backup

```bash
BACKUP_NAME="etcd-backup-$(date +%Y%m%d%H%M%S)"

kubectl apply -f - <<EOF
apiVersion: velero.io/v1
kind: Backup
metadata:
    name: ${BACKUP_NAME}
    namespace: openshift-adp
spec:
    includedNamespaces:
      - openshift-etcd
    storageLocation: backupstoragelocation
    ttl: 1h0m0s
    snapshotVolumes: false
EOF
```
