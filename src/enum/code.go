package enum

const (
	CodeOk = iota
	CodeErrMissingArgument
	CodeErrNotFound
	CodeErrBadArgument
	CodeErrDB
	CodeErrBlockchainNetworkError
	CodeErrCA
)

var CodeMessage = map[int]string {
	CodeOk:                 "success",
	CodeErrMissingArgument: "missing argument",
	CodeErrNotFound: 		"object not found",
	CodeErrBadArgument: 	"bad argument",
	CodeErrDB: 				"database error",
	CodeErrBlockchainNetworkError: "BlockchainNetworkError",
	CodeErrCA: "CA error",
}