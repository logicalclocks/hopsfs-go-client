package hdfs

/*
#include <stdlib.h>
*/
import "C"
import (
	"strings"
	"unsafe"
)

const hopsfsClientRemoteAccessEnabledEnv = "HOPSFS_CLIENT_REMOTE_ACCESS_ENABLED"

// remoteAccessEnabled reports whether the client should connect to the
// namenode(s) and datanodes via the external addresses they publish, as
// configured by the HOPSFS_CLIENT_REMOTE_ACCESS_ENABLED environment variable.
func remoteAccessEnabled() bool {
	return strings.EqualFold(getEnv(hopsfsClientRemoteAccessEnabledEnv), "true")
}

// getEnv bypasses Go's environment variables cache and reads directly from the
// OS. This enables changing env variables at runtime (delta-rs is a use case).
func getEnv(key string) string {
	ck := C.CString(key)
	defer C.free(unsafe.Pointer(ck))
	cp := C.getenv(ck)
	if cp == nil {
		return ""
	}
	return C.GoString(cp)
}
