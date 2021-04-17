package enum

const (
	// common
	StatusStarting 	= "starting"
	StatusRunning  	= "running"
	StatusError		= "error"

	// chaincode
	StatusUnpacking = "unpacking"
	StatusBuilding  = "building"

	// transaction
	StatusExecute	= "execute"
	StatusSuccess	= "success"
)
