package hdfs

import (
	"os"

	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"google.golang.org/protobuf/proto"
)

type RenameOptions uint32

const (
	RENAME_OPTION_NONE = 0
	RENAME_NOREPLACE   = 1 << 0
	RENAME_MOVETOTRASH = 1 << 1
)

// Rename renames (moves) a file.
func (c *Client) Rename(oldpath, newpath string) error {
	_, err := c.getFileInfo(newpath)
	err = interpretException(err)
	if err != nil && !os.IsNotExist(err) {
		return &os.PathError{"rename", newpath, err}
	}

	req := &hdfs.Rename2RequestProto{
		Src:           proto.String(oldpath),
		Dst:           proto.String(newpath),
		OverwriteDest: proto.Bool(true),
	}
	resp := &hdfs.Rename2ResponseProto{}

	err = c.leaderNamenode.Execute("rename2", req, resp)
	if err != nil {
		return &os.PathError{"rename", oldpath, interpretException(err)}
	}

	return nil
}

// Rename2 renames in one namenode call: the namenode checks the destination itself, so a
// destination that exists fails with os.ErrExist under RENAME_NOREPLACE, atomically.
func (c *Client) Rename2(oldpath, newpath string, options RenameOptions) error {
	req := &hdfs.Rename2RequestProto{
		Src:           proto.String(oldpath),
		Dst:           proto.String(newpath),
		OverwriteDest: proto.Bool(true),
	}

	if options&RENAME_NOREPLACE == RENAME_NOREPLACE {
		req.OverwriteDest = proto.Bool(false)
	} else {
		req.OverwriteDest = proto.Bool(true)
	}

	if options&RENAME_MOVETOTRASH == RENAME_MOVETOTRASH {
		req.MoveToTrash = proto.Bool(true)
	} else {
		req.MoveToTrash = proto.Bool(false)
	}

	resp := &hdfs.Rename2ResponseProto{}

	err := c.leaderNamenode.Execute("rename2", req, resp)
	if err != nil {
		return &os.PathError{"rename2", oldpath, interpretException(err)}
	}

	return nil
}
