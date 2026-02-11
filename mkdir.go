package hdfs

import (
	"os"
	"path"

	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"google.golang.org/protobuf/proto"
)

// Mkdir creates a new directory with the specified name and permission bits.
func (c *Client) Mkdir(dirname string, perm os.FileMode) error {
	return c.mkdir(dirname, perm, false)
}

// MkdirAll creates a directory for dirname, along with any necessary parents,
// and returns nil, or else returns an error. The permission bits perm are used
// for all directories that MkdirAll creates. If dirname is already a directory,
// MkdirAll does nothing and returns nil.
func (c *Client) MkdirAll(dirname string, perm os.FileMode) error {
	return c.mkdir(dirname, perm, true)
}

func (c *Client) mkdir(dirname string, perm os.FileMode, createParent bool) error {
	return c.mkdirWithGroup(dirname, perm, createParent, "")
}

func (c *Client) mkdirWithGroup(dirname string, perm os.FileMode, createParent bool, groupname string) error {
	dirname = path.Clean(dirname)

	info, err := c.getFileInfo(dirname)
	err = interpretException(err)
	if err == nil {
		if createParent && info.IsDir() {
			return nil
		}

		return &os.PathError{"mkdir", dirname, os.ErrExist}
	} else if !os.IsNotExist(err) {
		return &os.PathError{"mkdir", dirname, err}
	}

	req := &hdfs.MkdirsRequestProto{
		Src:          proto.String(dirname),
		Masked:       &hdfs.FsPermissionProto{Perm: proto.Uint32(uint32(perm))},
		CreateParent: proto.Bool(createParent),
	}

	// Set groupname if provided (non-empty)
	if groupname != "" {
		req.Groupname = proto.String(groupname)
	}

	resp := &hdfs.MkdirsResponseProto{}

	err = c.namenode.Execute("mkdirs", req, resp)
	if err != nil {
		return &os.PathError{"mkdir", dirname, interpretException(err)}
	}

	return nil
}

// MkdirWithGroup creates a new directory with the specified name, permission bits,
// and group name. If groupname is empty, the directory inherits the group from its parent.
func (c *Client) MkdirWithGroup(dirname string, perm os.FileMode, groupname string) error {
	return c.mkdirWithGroup(dirname, perm, false, groupname)
}

// MkdirAllWithGroup creates a directory for dirname, along with any necessary parents,
// and returns nil, or else returns an error. The permission bits perm are used for all
// directories that MkdirAllWithGroup creates. If groupname is non-empty, it is used for
// the created directories; otherwise they inherit the group from their parent.
func (c *Client) MkdirAllWithGroup(dirname string, perm os.FileMode, groupname string) error {
	return c.mkdirWithGroup(dirname, perm, true, groupname)
}
