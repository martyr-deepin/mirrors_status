package constants

type MirrorOperationStatus int

const (
	STATUS_FAILURE  MirrorOperationStatus = -1
	STATUS_WAITING  MirrorOperationStatus = iota
	STATUS_RUNNING                        = STATUS_WAITING + 1
	STATUS_FINISHED                       = STATUS_RUNNING + 1

	ADD    = "ADD"
	DELETE = "DELETE"
	SYNC   = "SYNC"
	SYNC_ALL = "SYNC ALL"
	SYNC_UPSTREAM = "SYNC UPSTREAM"
)