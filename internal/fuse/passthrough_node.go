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

type PassthroughNode struct {
	fs.Inode
	Path string
}

func (n *PassthroughNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
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

func (n *PassthroughNode) Lookup(ctx context.Context, name string, out *goFuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullPath := filepath.Join(n.Path, name)
	var stat syscall.Stat_t
	if err := syscall.Lstat(fullPath, &stat); err != nil {
		return nil, fs.ToErrno(err)
	}

	out.FromStat(&stat)

	var child *fs.Inode
	if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
		child = n.NewPersistentInode(ctx, &PassthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode})
	} else {
		child = n.NewPersistentInode(ctx, &PassthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode})
	}

	return child, 0
}

func (n *PassthroughNode) Mkdir(ctx context.Context, name string, mode uint32, out *goFuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullPath := filepath.Join(n.Path, name)
	if err := os.Mkdir(fullPath, os.FileMode(mode)); err != nil {
		return nil, fs.ToErrno(err)
	}

	var stat syscall.Stat_t
	if err := syscall.Lstat(fullPath, &stat); err != nil {
		return nil, fs.ToErrno(err)
	}
	out.FromStat(&stat)

	child := n.NewPersistentInode(ctx, &PassthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode})
	return child, 0
}

func (n *PassthroughNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	err := os.Remove(filepath.Join(n.Path, name))
	return fs.ToErrno(err)
}

func (n *PassthroughNode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	targetNode, ok := newParent.(*PassthroughNode)
	if !ok {
		return syscall.EXDEV
	}

	oldPath := filepath.Join(n.Path, name)
	newPath := filepath.Join(targetNode.Path, newName)
	err := os.Rename(oldPath, newPath)
	return fs.ToErrno(err)
}

func (n *PassthroughNode) Unlink(ctx context.Context, name string) syscall.Errno {
	err := os.Remove(filepath.Join(n.Path, name))
	return fs.ToErrno(err)
}

func (n *PassthroughNode) Getattr(ctx context.Context, f fs.FileHandle, out *goFuse.AttrOut) syscall.Errno {
	var stat syscall.Stat_t
	if err := syscall.Lstat(n.Path, &stat); err != nil {
		return fs.ToErrno(err)
	}
	out.FromStat(&stat)
	return 0
}

func (n *PassthroughNode) Setattr(ctx context.Context, f fs.FileHandle, in *goFuse.SetAttrIn, out *goFuse.AttrOut) syscall.Errno {
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

func (n *PassthroughNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *goFuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	fullPath := filepath.Join(n.Path, name)
	f, err := os.OpenFile(fullPath, int(flags)|os.O_CREATE, os.FileMode(mode))

	if err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}(f)

	var stat syscall.Stat_t
	if err := syscall.Lstat(fullPath, &stat); err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}
	out.FromStat(&stat)

	child := n.NewPersistentInode(ctx, &PassthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode})
	return child, &PassthroughFile{File: f}, 0, 0
}

func (n *PassthroughNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	f, err := os.OpenFile(n.Path, int(flags), 0)
	if err != nil {
		return nil, 0, fs.ToErrno(err)
	}
	return &PassthroughFile{File: f}, 0, 0
}

type PassthroughFile struct {
	File *os.File
}

func (f *PassthroughFile) Read(ctx context.Context, dest []byte, off int64) (goFuse.ReadResult, syscall.Errno) {
	n, err := f.File.ReadAt(dest, off)
	if err != nil && err != io.EOF {
		return nil, fs.ToErrno(err)
	}
	return goFuse.ReadResultData(dest[:n]), 0
}

func (f *PassthroughFile) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	n, err := f.File.WriteAt(data, off)
	return uint32(n), fs.ToErrno(err)
}

func (f *PassthroughFile) Flush(ctx context.Context) syscall.Errno {
	if err := f.File.Sync(); err != nil {
		return fs.ToErrno(err)
	}
	return 0
}

func (f *PassthroughFile) Release(ctx context.Context) syscall.Errno {
	if err := f.File.Close(); err != nil {
		return fs.ToErrno(err)
	}
	return 0
}

var _ = (fs.NodeReaddirer)((*PassthroughNode)(nil))
var _ = (fs.NodeLookuper)((*PassthroughNode)(nil))
var _ = (fs.NodeMkdirer)((*PassthroughNode)(nil))
var _ = (fs.NodeRmdirer)((*PassthroughNode)(nil))
var _ = (fs.NodeRenamer)((*PassthroughNode)(nil))
var _ = (fs.NodeUnlinker)((*PassthroughNode)(nil))
var _ = (fs.NodeGetattrer)((*PassthroughNode)(nil))
var _ = (fs.NodeSetattrer)((*PassthroughNode)(nil))
var _ = (fs.NodeCreater)((*PassthroughNode)(nil))
var _ = (fs.NodeOpener)((*PassthroughNode)(nil))

var _ = (fs.FileReader)((*PassthroughFile)(nil))
var _ = (fs.FileWriter)((*PassthroughFile)(nil))
var _ = (fs.FileFlusher)((*PassthroughFile)(nil))
var _ = (fs.FileReleaser)((*PassthroughFile)(nil))
