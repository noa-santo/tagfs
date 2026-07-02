package fuse

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	goFuse "github.com/hanwen/go-fuse/v2/fuse"
)

var passthroughLogger = log.New(os.Stdout, "PASSTHROUGH NODE: ", log.LstdFlags|log.Lmicroseconds)

type passthroughNode struct {
	fs.Inode
	Path string
}

func (n *passthroughNode) Readdir(_ context.Context) (fs.DirStream, syscall.Errno) {
	entries, err := os.ReadDir(n.Path)
	if err != nil {
		return nil, fs.ToErrno(err)
	}

	var result []goFuse.DirEntry
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		result = append(result, goFuse.DirEntry{
			Name: entry.Name(),
			Ino:  0,
			Mode: uint32(info.Mode()),
		})
	}
	return fs.NewListDirStream(result), 0
}

func (n *passthroughNode) Lookup(ctx context.Context, name string, out *goFuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullPath := filepath.Join(n.Path, name)
	var stat syscall.Stat_t
	if err := syscall.Lstat(fullPath, &stat); err != nil {
		return nil, fs.ToErrno(err)
	}

	out.FromStat(&stat)

	var child *fs.Inode
	if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
		child = n.NewPersistentInode(ctx, &passthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode})
	} else {
		child = n.NewPersistentInode(ctx, &passthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode})
	}

	return child, 0
}

func (n *passthroughNode) Mkdir(ctx context.Context, name string, mode uint32, out *goFuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullPath := filepath.Join(n.Path, name)
	if err := os.Mkdir(fullPath, os.FileMode(mode)); err != nil {
		return nil, fs.ToErrno(err)
	}

	var stat syscall.Stat_t
	if err := syscall.Lstat(fullPath, &stat); err != nil {
		return nil, fs.ToErrno(err)
	}
	out.FromStat(&stat)

	child := n.NewPersistentInode(ctx, &passthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode})
	return child, 0
}

func (n *passthroughNode) Rmdir(_ context.Context, name string) syscall.Errno {
	err := os.Remove(filepath.Join(n.Path, name))
	return fs.ToErrno(err)
}

func (n *passthroughNode) Rename(_ context.Context, name string, newParent fs.InodeEmbedder, newName string, _ uint32) syscall.Errno {
	targetNode, ok := newParent.(*passthroughNode)
	if !ok {
		return syscall.EXDEV
	}

	oldPath := filepath.Join(n.Path, name)
	newPath := filepath.Join(targetNode.Path, newName)
	err := os.Rename(oldPath, newPath)
	return fs.ToErrno(err)
}

func (n *passthroughNode) Unlink(_ context.Context, name string) syscall.Errno {
	err := os.Remove(filepath.Join(n.Path, name))
	return fs.ToErrno(err)
}

func (n *passthroughNode) Getattr(_ context.Context, _ fs.FileHandle, out *goFuse.AttrOut) syscall.Errno {
	var stat syscall.Stat_t
	if err := syscall.Lstat(n.Path, &stat); err != nil {
		return fs.ToErrno(err)
	}
	out.FromStat(&stat)
	return 0
}

func (n *passthroughNode) Setattr(ctx context.Context, f fs.FileHandle, in *goFuse.SetAttrIn, out *goFuse.AttrOut) syscall.Errno {
	if sz, ok := in.GetSize(); ok {
		if err := os.Truncate(n.Path, int64(sz)); err != nil {
			return fs.ToErrno(err)
		}
	}
	if mode, ok := in.GetMode(); ok {
		if err := os.Chmod(n.Path, os.FileMode(mode)); err != nil {
			return fs.ToErrno(err)
		}
	}
	uid, uok := in.GetUID()
	gid, gok := in.GetGID()
	if uok || gok {
		suid := -1
		sgid := -1
		if uok {
			suid = int(uid)
		}
		if gok {
			sgid = int(gid)
		}
		if err := os.Chown(n.Path, suid, sgid); err != nil {
			return fs.ToErrno(err)
		}
	}

	return n.Getattr(ctx, f, out)
}

func (n *passthroughNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *goFuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	fullPath := filepath.Join(n.Path, name)
	f, err := os.OpenFile(fullPath, int(flags)|os.O_CREATE, os.FileMode(mode))

	if err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			passthroughLogger.Printf("Error closing file: %v", err)
		}
	}(f)

	var stat syscall.Stat_t
	if err := syscall.Lstat(fullPath, &stat); err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}
	out.FromStat(&stat)

	child := n.NewPersistentInode(ctx, &passthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode})
	return child, &passthroughFile{File: f}, 0, 0
}

func (n *passthroughNode) Open(_ context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	f, err := os.OpenFile(n.Path, int(flags), 0)
	if err != nil {
		return nil, 0, fs.ToErrno(err)
	}
	return &passthroughFile{File: f}, 0, 0
}

type passthroughFile struct {
	File *os.File
}

func (f *passthroughFile) Read(_ context.Context, dest []byte, off int64) (goFuse.ReadResult, syscall.Errno) {
	n, err := f.File.ReadAt(dest, off)
	if err != nil && err != io.EOF {
		return nil, fs.ToErrno(err)
	}
	return goFuse.ReadResultData(dest[:n]), 0
}

func (f *passthroughFile) Write(_ context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	n, err := f.File.WriteAt(data, off)
	return uint32(n), fs.ToErrno(err)
}

func (f *passthroughFile) Flush(_ context.Context) syscall.Errno {
	if err := f.File.Sync(); err != nil {
		return fs.ToErrno(err)
	}
	return 0
}

func (f *passthroughFile) Release(_ context.Context) syscall.Errno {
	if err := f.File.Close(); err != nil {
		return fs.ToErrno(err)
	}
	return 0
}

var _ = (fs.NodeReaddirer)((*passthroughNode)(nil))
var _ = (fs.NodeLookuper)((*passthroughNode)(nil))
var _ = (fs.NodeMkdirer)((*passthroughNode)(nil))
var _ = (fs.NodeRmdirer)((*passthroughNode)(nil))
var _ = (fs.NodeRenamer)((*passthroughNode)(nil))
var _ = (fs.NodeUnlinker)((*passthroughNode)(nil))
var _ = (fs.NodeGetattrer)((*passthroughNode)(nil))
var _ = (fs.NodeSetattrer)((*passthroughNode)(nil))
var _ = (fs.NodeCreater)((*passthroughNode)(nil))
var _ = (fs.NodeOpener)((*passthroughNode)(nil))

var _ = (fs.FileReader)((*passthroughFile)(nil))
var _ = (fs.FileWriter)((*passthroughFile)(nil))
var _ = (fs.FileFlusher)((*passthroughFile)(nil))
var _ = (fs.FileReleaser)((*passthroughFile)(nil))
