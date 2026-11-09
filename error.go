package hdfs

import (
	"os"
	"strings"
	"syscall"
)

const (
	fileNotFoundException        = "java.io.FileNotFoundException"
	permissionDeniedException    = "org.apache.hadoop.security.AccessControlException"
	pathIsNotEmptyDirException   = "org.apache.hadoop.fs.PathIsNotEmptyDirectoryException"
	fileAlreadyExistsException   = "org.apache.hadoop.fs.FileAlreadyExistsException"
	alreadyBeingCreatedException = "org.apache.hadoop.hdfs.protocol.AlreadyBeingCreatedException"
	invalidPathException         = "org.apache.hadoop.fs.InvalidPathException"
	safeModeException            = "org.apache.hadoop.hdfs.server.namenode.SafeModeException"
	dSQuotaExceededException     = "org.apache.hadoop.hdfs.protocol.DSQuotaExceededException"
	nSQuotaExceededException     = "org.apache.hadoop.hdfs.protocol.NSQuotaExceededException"
	parentNotDirectoryException  = "org.apache.hadoop.fs.ParentNotDirectoryException"
	unresolvedLinkException      = "org.apache.hadoop.fs.UnresolvedLinkException"
	notReplicatedYetException    = "org.apache.hadoop.hdfs.server.namenode.NotReplicatedYetException"
	illegalArgumentException     = "org.apache.hadoop.HadoopIllegalArgumentException"
	notALeaderException          = "org.apache.hadoop.ipc.NotALeaderException"
	retriableException           = "org.apache.hadoop.ipc.RetriableException"
	subtreeLockedException       = "org.apache.hadoop.ipc.SubtreeLockedException"
	javaIOException              = "java.io.IOException"
)

// Error represents a remote java exception from an HDFS namenode or datanode.
type Error interface {
	// Method returns the RPC method that encountered an error.
	Method() string
	// Desc returns the long form of the error code (for example ERROR_CHECKSUM).
	Desc() string
	// Exception returns the java exception class name (for example
	// java.io.FileNotFoundException).
	Exception() string
	// Message returns the full error message, complete with java exception
	// traceback.
	Message() string
}

func interpretCreateException(err error) error {
	if remoteErr, ok := err.(Error); ok && remoteErr.Exception() == alreadyBeingCreatedException {
		return os.ErrExist
	}

	return interpretException(err)
}

// unwrapping exceptions as in hopsfs all exceptions
// are wrapped in IOException
func unwrapHopsFSException(err error) string {
	if remoteErr, ok := err.(Error); ok {
		exception := remoteErr.Exception()
		message := remoteErr.Message()

		if strings.HasPrefix(exception, javaIOException) {
			if strings.HasPrefix(message, fileNotFoundException) {
				return fileNotFoundException
			} else if strings.HasPrefix(message, permissionDeniedException) {
				return permissionDeniedException
			} else if strings.HasPrefix(message, pathIsNotEmptyDirException) {
				return pathIsNotEmptyDirException
			} else if strings.HasPrefix(message, fileAlreadyExistsException) {
				return fileAlreadyExistsException
			} else if strings.HasPrefix(message, alreadyBeingCreatedException) {
				return alreadyBeingCreatedException
			} else if strings.HasPrefix(message, invalidPathException) {
				return invalidPathException
			} else if strings.HasPrefix(message, safeModeException) {
				return safeModeException
			} else if strings.HasPrefix(message, dSQuotaExceededException) {
				return dSQuotaExceededException
			} else if strings.HasPrefix(message, nSQuotaExceededException) {
				return nSQuotaExceededException
			} else if strings.HasPrefix(message, parentNotDirectoryException) {
				return parentNotDirectoryException
			} else if strings.HasPrefix(message, unresolvedLinkException) {
				return unresolvedLinkException
			} else if strings.HasPrefix(message, notReplicatedYetException) {
				return notReplicatedYetException
			} else if strings.HasPrefix(message, illegalArgumentException) {
				return illegalArgumentException
			} else if strings.HasPrefix(message, notALeaderException) {
				return notALeaderException
			}
		} else {
			return exception
		}
	}

	return "Not a Remote HopsFS Exception"
}

func interpretException(err error) error {
	var exception string
	if _, ok := err.(Error); ok {
		exception = unwrapHopsFSException(err)
	}

	switch exception {
	case fileNotFoundException:
		return os.ErrNotExist
	case permissionDeniedException:
		return os.ErrPermission
	case pathIsNotEmptyDirException:
		return syscall.ENOTEMPTY
	case fileAlreadyExistsException:
		return os.ErrExist
	case invalidPathException:
		return syscall.ENOENT
	case safeModeException:
		return syscall.EROFS
	case dSQuotaExceededException:
		return syscall.EDQUOT
	case nSQuotaExceededException:
		return syscall.EDQUOT
	case parentNotDirectoryException:
		return syscall.ENOENT
	case unresolvedLinkException:
		return syscall.ENOLINK
	case notReplicatedYetException:
		return syscall.EPROTO // Protocol Error
	case illegalArgumentException:
		return os.ErrInvalid
	case javaIOException:
		// In HopsFS all RuntimeExceptions are
		// caught by the TX request handler and
		// then thrown as IOException.
		return syscall.EIO
	case notALeaderException:
		return syscall.EAGAIN
	case retriableException:
		// Execute's retries are spent. A path under a subtree operation is
		// the namenode's answer over a healthy connection, so it gets an
		// errno of its own instead of passing as an unknown failure.
		if strings.HasPrefix(err.(Error).Message(), subtreeLockedException+":") {
			return syscall.EBUSY
		}
		return err
	default:
		return err
	}
}
