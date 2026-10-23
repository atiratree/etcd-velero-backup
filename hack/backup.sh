#!/usr/bin/env bash

BACKUP_NAME="${BACKUP_NAME:-etcd-backup-$(date +%Y%m%d%H%M%S)}"

kubectl apply -f - <<EOF
apiVersion: velero.io/v1
kind: Backup
metadata:
  name: ${BACKUP_NAME}
  namespace: openshift-etcd
spec:
  includedNamespaces:
    - openshift-etcd
  storageLocation: default
  ttl: 1h0m0s
  snapshotVolumes: false
  defaultVolumesToFsBackup: false
EOF
