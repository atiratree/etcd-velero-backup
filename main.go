package main

import (
	"github.com/vmware-tanzu/velero/pkg/plugin/framework"

	"github.com/atiratree/etcd-backup/internal/plugin"
)

func main() {
	framework.NewServer().
		RegisterBackupItemAction("etcd-backup.io/backup-plugin", plugin.NewBackupPlugin).
		RegisterObjectStore("etcd-backup.io/local-store", plugin.NewFileObjectStore).
		//RegisterRestoreItemAction("etcd-backup.io/restore-plugin", plugin.NewRestorePlugin).
		Serve()
}
