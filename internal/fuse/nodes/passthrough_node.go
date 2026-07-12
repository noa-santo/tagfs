package nodes

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	goFuse "github.com/hanwen/go-fuse/v2/fuse"
	"golang.org/x/sys/unix"
)

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
	return fs.NewListDirStream(result), fs.OK
}

func (n *passthroughNode) Lookup(ctx context.Context, name string, out *goFuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullPath := filepath.Join(n.Path, name)
	var stat syscall.Stat_t
	if err := syscall.Lstat(fullPath, &stat); err != nil {
		return nil, fs.ToErrno(err)
	}
	out.FromStat(&stat)
	return n.NewPersistentInode(ctx, &passthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode}), 0
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

	return n.NewPersistentInode(ctx, &passthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode}), 0
}

func (n *passthroughNode) Rmdir(_ context.Context, name string) syscall.Errno {
	err := os.Remove(filepath.Join(n.Path, name))
	return fs.ToErrno(err)
}

func (n *passthroughNode) Rename(_ context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	targetNode, ok := newParent.(*passthroughNode)
	if !ok {
		return syscall.EXDEV
	}

	oldPath := filepath.Join(n.Path, name)
	newPath := filepath.Join(targetNode.Path, newName)

	if flags != 0 {
		err := unix.Renameat2(unix.AT_FDCWD, oldPath, unix.AT_FDCWD, newPath, uint(flags))
		return fs.ToErrno(err)
	}

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
		suid, sgid := -1, -1
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
	atime, aok := in.GetATime()
	mtime, mok := in.GetMTime()
	if aok || mok {
		if !aok || !mok {
			var st syscall.Stat_t
			if err := syscall.Lstat(n.Path, &st); err != nil {
				return fs.ToErrno(err)
			}
			if !aok {
				atime = time.Unix(st.Atim.Sec, st.Atim.Nsec)
			}
			if !mok {
				mtime = time.Unix(st.Mtim.Sec, st.Mtim.Nsec)
			}
		}
		if err := os.Chtimes(n.Path, atime, mtime); err != nil {
			return fs.ToErrno(err)
		}
	}

	return n.Getattr(ctx, f, out)
}

func (n *passthroughNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *goFuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	fullPath := filepath.Join(n.Path, name)
	//goland:noinspection GoResourceLeak file isn't closed because that will be handled by the process that created the file
	f, err := os.OpenFile(fullPath, int(flags)|os.O_CREATE, os.FileMode(mode))
	if err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}

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

func (n *passthroughNode) Symlink(ctx context.Context, target, name string, out *goFuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullPath := filepath.Join(n.Path, name)
	if err := os.Symlink(target, fullPath); err != nil {
		return nil, fs.ToErrno(err)
	}

	var stat syscall.Stat_t
	if err := syscall.Lstat(fullPath, &stat); err != nil {
		return nil, fs.ToErrno(err)
	}
	out.FromStat(&stat)
	return n.NewPersistentInode(ctx, &passthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode}), 0
}

func (n *passthroughNode) Readlink(_ context.Context) ([]byte, syscall.Errno) {
	target, err := os.Readlink(n.Path)
	if err != nil {
		return nil, fs.ToErrno(err)
	}
	return []byte(target), 0
}

func (n *passthroughNode) Link(ctx context.Context, target fs.InodeEmbedder, name string, out *goFuse.EntryOut) (*fs.Inode, syscall.Errno) {
	targetNode, ok := target.(*passthroughNode)
	if !ok {
		return nil, syscall.EXDEV
	}

	fullPath := filepath.Join(n.Path, name)
	if err := os.Link(targetNode.Path, fullPath); err != nil {
		return nil, fs.ToErrno(err)
	}

	var stat syscall.Stat_t
	if err := syscall.Lstat(fullPath, &stat); err != nil {
		return nil, fs.ToErrno(err)
	}
	out.FromStat(&stat)
	return n.NewPersistentInode(ctx, &passthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode}), 0
}

func (n *passthroughNode) Mknod(ctx context.Context, name string, mode, dev uint32, out *goFuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullPath := filepath.Join(n.Path, name)
	if err := unix.Mknod(fullPath, mode, int(dev)); err != nil {
		return nil, fs.ToErrno(err)
	}

	var stat syscall.Stat_t
	if err := syscall.Lstat(fullPath, &stat); err != nil {
		return nil, fs.ToErrno(err)
	}
	out.FromStat(&stat)
	return n.NewPersistentInode(ctx, &passthroughNode{Path: fullPath}, fs.StableAttr{Mode: stat.Mode}), 0
}

func (n *passthroughNode) Statfs(_ context.Context, out *goFuse.StatfsOut) syscall.Errno {
	var stat unix.Statfs_t
	if err := unix.Statfs(n.Path, &stat); err != nil {
		return fs.ToErrno(err)
	}
	out.Blocks = stat.Blocks
	out.Bfree = stat.Bfree
	out.Bavail = stat.Bavail
	out.Files = stat.Files
	out.Ffree = stat.Ffree
	out.Bsize = uint32(stat.Bsize)
	out.NameLen = uint32(stat.Namelen)
	out.Frsize = uint32(stat.Frsize)
	return 0
}

func (n *passthroughNode) Getxattr(_ context.Context, attr string, dest []byte) (uint32, syscall.Errno) {
	sz, err := unix.Lgetxattr(n.Path, attr, dest)
	if err != nil {
		return 0, fs.ToErrno(err)
	}
	return uint32(sz), 0
}

func (n *passthroughNode) Setxattr(_ context.Context, attr string, data []byte, flags uint32) syscall.Errno {
	err := unix.Lsetxattr(n.Path, attr, data, int(flags))
	return fs.ToErrno(err)
}

func (n *passthroughNode) Removexattr(_ context.Context, attr string) syscall.Errno {
	err := unix.Lremovexattr(n.Path, attr)
	return fs.ToErrno(err)
}

func (n *passthroughNode) Listxattr(_ context.Context, dest []byte) (uint32, syscall.Errno) {
	sz, err := unix.Llistxattr(n.Path, dest)
	if err != nil {
		return 0, fs.ToErrno(err)
	}
	return uint32(sz), 0
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

func (f *passthroughFile) Fsync(_ context.Context, _ uint32) syscall.Errno {
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

func (f *passthroughFile) Allocate(_ context.Context, off uint64, size uint64, mode uint32) syscall.Errno {
	err := unix.Fallocate(int(f.File.Fd()), mode, int64(off), int64(size))
	return fs.ToErrno(err)
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
var _ = (fs.NodeSymlinker)((*passthroughNode)(nil))
var _ = (fs.NodeReadlinker)((*passthroughNode)(nil))
var _ = (fs.NodeLinker)((*passthroughNode)(nil))
var _ = (fs.NodeMknoder)((*passthroughNode)(nil))
var _ = (fs.NodeStatfser)((*passthroughNode)(nil))
var _ = (fs.NodeGetxattrer)((*passthroughNode)(nil))
var _ = (fs.NodeSetxattrer)((*passthroughNode)(nil))
var _ = (fs.NodeRemovexattrer)((*passthroughNode)(nil))
var _ = (fs.NodeListxattrer)((*passthroughNode)(nil))

var _ = (fs.FileReader)((*passthroughFile)(nil))
var _ = (fs.FileWriter)((*passthroughFile)(nil))
var _ = (fs.FileFlusher)((*passthroughFile)(nil))
var _ = (fs.FileReleaser)((*passthroughFile)(nil))
var _ = (fs.FileFsyncer)((*passthroughFile)(nil))
var _ = (fs.FileAllocater)((*passthroughFile)(nil))
