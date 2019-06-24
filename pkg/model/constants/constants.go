package constants

type MirrorOperationStatus int

const (
	STATUS_ABORTED  MirrorOperationStatus = -2
	STATUS_FAILURE  MirrorOperationStatus = -1
	STATUS_WAITING  MirrorOperationStatus = 0
	STATUS_RUNNING  MirrorOperationStatus = 1
	STATUS_FINISHED MirrorOperationStatus = 2

	ADD           = "ADD"
	DELETE        = "DELETE"
	SYNC          = "SYNC"
	SYNC_ALL      = "SYNC ALL"
	SYNC_UPSTREAM = "SYNC UPSTREAM"
)
