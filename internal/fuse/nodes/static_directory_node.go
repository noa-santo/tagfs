package nodes

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/noa-santo/tagfs/internal/config"
	"github.com/noa-santo/tagfs/internal/db"
	"github.com/noa-santo/tagfs/internal/db/gen"
	"github.com/noa-santo/tagfs/internal/logic"
	"github.com/oklog/ulid/v2"
)

var staticDirLogger = log.New(os.Stdout, "STATIC_DIR: ", log.LstdFlags)

type staticDirectoryNode struct {
	fs.Inode
	nodeConfig config.DirectoryConfig
}

func (n *staticDirectoryNode) isSubdir(name string) bool {
	for _, subdir := range n.nodeConfig.Subdirectories {
		if subdir.Name == name {
			return true
		}
	}
	return false
}

func (n *staticDirectoryNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Readdir: GetNodesForDir failed for %s: %v", n.nodeConfig.Name, err)
		return nil, syscall.EIO
	}

	result := make([]fuse.DirEntry, 0, len(nodes)+len(n.nodeConfig.Subdirectories))

	for _, child := range n.nodeConfig.Subdirectories {
		name := child.Name
		if n.isSubdir(name) {
			result = append(result, fuse.DirEntry{
				Name: name,
				Ino:  0,
				Mode: uint32(syscall.S_IFDIR | 0755),
			})
		}
	}
	for _, node := range nodes {
		result = append(result, fuse.DirEntry{
			Name: node.OrigName,
			Ino:  0,
			Mode: uint32(node.Mode),
		})
	}
	return fs.NewListDirStream(result), fs.OK
}

func (n *staticDirectoryNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	if child := n.GetChild(name); child != nil {
		return child, fs.OK
	}

	for _, subdir := range n.nodeConfig.Subdirectories {
		if subdir.Name == name {
			childNode := &staticDirectoryNode{
				nodeConfig: subdir,
			}
			childInode := n.NewInode(ctx, childNode, fs.StableAttr{
				Mode: uint32(syscall.S_IFDIR | 0755),
			})
			return childInode, fs.OK
		}
	}

	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Lookup: GetNodesForDir failed for %s: %v", n.nodeConfig.Name, err)
		return nil, syscall.EIO
	}

	for _, node := range nodes {
		if node.OrigName != name {
			continue
		}

		physicalPath := filepath.Join(config.Get().StoragePath, ".data", node.ID, node.OrigName)

		var st syscall.Stat_t
		if err := syscall.Stat(physicalPath, &st); err != nil {
			staticDirLogger.Printf("Lookup: stat failed for %s (id=%s): %v", physicalPath, node.ID, err)
			return nil, fs.ToErrno(err)
		}
		out.Attr.FromStat(&st)

		childNode := &passthroughNode{Path: physicalPath}
		childInode := n.NewInode(ctx, childNode, fs.StableAttr{
			Mode: uint32(node.Mode) & syscall.S_IFMT,
		})
		return childInode, fs.OK
	}

	return nil, syscall.ENOENT
}

func (n *staticDirectoryNode) Mkdir(ctx context.Context, name string, mode uint32, _ *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	if !n.nodeConfig.Rules.AllowSubdirCreation {
		return nil, syscall.EPERM
	}
	if n.nodeConfig.Rules.ForceNamePattern {
		if !logic.MatchesNamePattern(name, n.nodeConfig.Rules.NamePatterns) {
			return nil, syscall.EINVAL
		}
	}
	if child := n.GetChild(name); child != nil || n.isSubdir(name) {
		return nil, syscall.EEXIST
	}
	staticDirLogger.Printf("Ingesting new directory at %s: %s", n.nodeConfig.Name, name)
	dirID := ulid.Make().String()
	dataPath := filepath.Join(config.Get().StoragePath, ".data", dirID)
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fs.ToErrno(err)
	}
	physicalPath := filepath.Join(dataPath, name)
	if err := os.Mkdir(physicalPath, os.FileMode(mode)); err != nil {
		return nil, fs.ToErrno(err)
	}
	err := db.Get().Queries.InsertNode(ctx, gen.InsertNodeParams{
		ID:       dirID,
		OrigName: name,
		Mode:     logic.ToStoredMode(mode, true),
	})
	if err != nil {
		staticDirLogger.Printf("Error inserting directory into DB: %v", err)
		return nil, syscall.EIO
	}

	err = db.Get().UpdateTags(dirID, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Error updating tags for directory %s: %v", dirID, err)
		return nil, syscall.EIO
	}

	childNode := &passthroughNode{Path: physicalPath}
	childInode := n.NewPersistentInode(ctx, childNode, fs.StableAttr{Mode: syscall.S_IFDIR | mode})

	return childInode, fs.OK
}

func (n *staticDirectoryNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	if n.isSubdir(name) {
		return syscall.EPERM
	}
	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Rmdir: GetNodesForDir failed for %s: %v", n.nodeConfig.Name, err)
		return syscall.EIO
	}
	for _, node := range nodes {
		if node.OrigName != name {
			continue
		}
		physicalPath := filepath.Join(config.Get().StoragePath, ".data", node.ID)
		if err := os.RemoveAll(physicalPath); err != nil {
			staticDirLogger.Printf("Rmdir: RemoveAll failed for %s: %v", physicalPath, err)
			return syscall.EIO
		}
		err := db.Get().Queries.DeleteNode(ctx, node.ID)
		if err != nil {
			return fs.ToErrno(err)
		}
		return fs.OK
	}
	return syscall.ENOENT
}

func (n *staticDirectoryNode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	if flags != 0 {
		// RENAME_NOREPLACE / RENAME_EXCHANGE aren't supported by the tag-based store yet.
		return syscall.ENOSYS
	}
	if n.isSubdir(name) {
		return syscall.EPERM
	}

	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Rename: GetNodesForDir failed for %s: %v", n.nodeConfig.Name, err)
		return syscall.EIO
	}

	var target gen.Node
	found := false
	for _, node := range nodes {
		if node.OrigName == name {
			target = node
			found = true
			break
		}
	}
	if !found {
		return syscall.ENOENT
	}

	newParentNode, ok := newParent.(*staticDirectoryNode)
	if !ok {
		staticDirLogger.Printf("Rename: unsupported destination type for %s -> %s", name, newName)
		return syscall.EXDEV
	}
	if newParentNode.GetChild(newName) != nil {
		return syscall.EEXIST
	}

	oldPhysicalPath := filepath.Join(config.Get().StoragePath, ".data", target.ID, target.OrigName)
	newPhysicalPath := filepath.Join(config.Get().StoragePath, ".data", target.ID, newName)
	if oldPhysicalPath != newPhysicalPath {
		if err := os.Rename(oldPhysicalPath, newPhysicalPath); err != nil {
			staticDirLogger.Printf("Rename: physical rename failed %s -> %s: %v", oldPhysicalPath, newPhysicalPath, err)
			return fs.ToErrno(err)
		}
	}

	if err := db.Get().Queries.RenameNode(ctx, gen.RenameNodeParams{
		ID:       target.ID,
		OrigName: newName,
	}); err != nil {
		staticDirLogger.Printf("Rename: DB rename failed for %s: %v", target.ID, err)
		if oldPhysicalPath != newPhysicalPath {
			_ = os.Rename(newPhysicalPath, oldPhysicalPath)
		}
		return syscall.EIO
	}

	if newParentNode.nodeConfig.Name != n.nodeConfig.Name {
		tags := append(newParentNode.nodeConfig.Tags, logic.GetImplicitTags(newParentNode.nodeConfig.Tags)...)
		if err := db.Get().UpdateTags(target.ID, tags); err != nil {
			staticDirLogger.Printf("Rename: UpdateTags failed for %s: %v", target.ID, err)
			return syscall.EIO
		}
	}

	return fs.OK
}

func (n *staticDirectoryNode) Create(ctx context.Context, name string, flags uint32, mode uint32, _ *fuse.EntryOut) (node *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	if !n.nodeConfig.Rules.AllowFileCreation {
		return nil, nil, 0, syscall.EPERM
	}
	if n.nodeConfig.Rules.ForceNamePattern {
		if !logic.MatchesNamePattern(name, n.nodeConfig.Rules.NamePatterns) {
			return nil, nil, 0, syscall.EINVAL
		}
	}
	staticDirLogger.Printf("Ingesting new file at %s: %s", n.nodeConfig.Name, name)
	fileID := ulid.Make().String()
	physicalPath := filepath.Join(config.Get().StoragePath, ".data", fileID, name)
	if err := os.MkdirAll(filepath.Dir(physicalPath), 0755); err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}
	f, err := os.OpenFile(physicalPath, int(flags)|os.O_CREATE, os.FileMode(mode))
	if err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}
	err = db.Get().Queries.InsertNode(ctx, gen.InsertNodeParams{
		ID:       fileID,
		OrigName: name,
		Mode:     logic.ToStoredMode(mode, false),
	})
	if err != nil {
		if closeErr := f.Close(); closeErr != nil {
			rootLogger.Printf("Error closing physical file: %v", closeErr)
			return nil, nil, 0, 0
		}
		if rmErr := os.Remove(physicalPath); rmErr != nil {
			rootLogger.Printf("Error removing physical file: %v", rmErr)
			return nil, nil, 0, 0
		}
		rootLogger.Printf("Error inserting file into DB: %v", err)
		return nil, nil, 0, syscall.EIO
	}

	if err := db.Get().UpdateTags(fileID, n.nodeConfig.Tags); err != nil {
		staticDirLogger.Printf("Create: UpdateTags failed for %s: %v", fileID, err)
	}

	childNode := &passthroughNode{Path: physicalPath}
	childInode := n.NewPersistentInode(ctx, childNode, fs.StableAttr{Mode: mode})
	return childInode, f, 0, fs.OK
}

func (n *staticDirectoryNode) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Open: GetNodesForDir failed for %s: %v", n.nodeConfig.Name, err)
		return nil, 0, syscall.EIO
	}
	for _, node := range nodes {
		if node.OrigName == n.nodeConfig.Name {
			physicalPath := filepath.Join(config.Get().StoragePath, ".data", node.ID, node.OrigName)
			f, err := os.OpenFile(physicalPath, int(flags), 0)
			if err != nil {
				return nil, 0, fs.ToErrno(err)
			}
			return &passthroughFile{File: f}, 0, fs.OK
		}
	}
	return nil, 0, syscall.ENOENT
}

func (n *staticDirectoryNode) Opendir(context.Context) syscall.Errno {
	return fs.OK
}

func (n *staticDirectoryNode) Access(context.Context, uint32) syscall.Errno {
	return fs.OK
}

func (n *staticDirectoryNode) Unlink(ctx context.Context, name string) syscall.Errno {
	if n.isSubdir(name) {
		return syscall.EPERM
	}
	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Unlink: GetNodesForDir failed for %s: %v", n.nodeConfig.Name, err)
		return syscall.EIO
	}
	for _, node := range nodes {
		if node.OrigName == name {
			physicalPath := filepath.Join(config.Get().StoragePath, ".data", node.ID)
			if err := os.RemoveAll(physicalPath); err != nil {
				return fs.ToErrno(err)
			}
			if err := db.Get().Queries.DeleteNode(ctx, node.ID); err != nil {
				return fs.ToErrno(err)
			}
			return fs.OK
		}
	}
	return syscall.ENOENT
}

func (n *staticDirectoryNode) Symlink(ctx context.Context, target, name string, _ *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	if !n.nodeConfig.Rules.AllowFileCreation {
		return nil, syscall.EPERM
	}
	if n.GetChild(name) != nil || n.isSubdir(name) {
		return nil, syscall.EEXIST
	}
	if n.nodeConfig.Rules.ForceNamePattern {
		if !logic.MatchesNamePattern(name, n.nodeConfig.Rules.NamePatterns) {
			return nil, syscall.EINVAL
		}
	}
	linkID := ulid.Make().String()
	physicalPath := filepath.Join(config.Get().StoragePath, ".data", linkID, name)
	if err := os.MkdirAll(filepath.Dir(physicalPath), 0755); err != nil {
		return nil, fs.ToErrno(err)
	}
	if err := os.Symlink(target, physicalPath); err != nil {
		return nil, fs.ToErrno(err)
	}

	var st syscall.Stat_t
	if err := syscall.Lstat(physicalPath, &st); err != nil {
		return nil, fs.ToErrno(err)
	}

	err := db.Get().Queries.InsertNode(ctx, gen.InsertNodeParams{
		ID:       linkID,
		OrigName: name,
		Mode:     logic.ToStoredMode(uint32(syscall.S_IFLNK|0777), false),
	})
	if err != nil {
		_ = os.Remove(physicalPath)
		staticDirLogger.Printf("Symlink: InsertNode failed for %s: %v", name, err)
		return nil, syscall.EIO
	}
	if err := db.Get().UpdateTags(linkID, n.nodeConfig.Tags); err != nil {
		staticDirLogger.Printf("Symlink: UpdateTags failed for %s: %v", linkID, err)
	}

	childNode := &passthroughNode{Path: physicalPath}
	childInode := n.NewPersistentInode(ctx, childNode, fs.StableAttr{Mode: syscall.S_IFLNK})
	return childInode, fs.OK
}

func (n *staticDirectoryNode) Link(context.Context, fs.InodeEmbedder, string, *fuse.EntryOut) (node *fs.Inode, errno syscall.Errno) {
	return nil, syscall.EPERM
}

func (n *staticDirectoryNode) Mknod(_ context.Context, _ string, _, _ uint32, _ *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	return nil, syscall.EPERM
}

func (n *staticDirectoryNode) Setattr(context.Context, fs.FileHandle, *fuse.SetAttrIn, *fuse.AttrOut) syscall.Errno {
	return syscall.EPERM
}

func (n *staticDirectoryNode) Getattr(_ context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = syscall.S_IFDIR | 0755
	return fs.OK
}

var _ fs.NodeReaddirer = (*staticDirectoryNode)(nil)
var _ fs.NodeLookuper = (*staticDirectoryNode)(nil)
var _ fs.NodeMkdirer = (*staticDirectoryNode)(nil)
var _ fs.NodeRmdirer = (*staticDirectoryNode)(nil)
var _ fs.NodeRenamer = (*staticDirectoryNode)(nil)
var _ fs.NodeCreater = (*staticDirectoryNode)(nil)
var _ fs.NodeOpener = (*staticDirectoryNode)(nil)
var _ fs.NodeOpendirer = (*staticDirectoryNode)(nil)
var _ fs.NodeAccesser = (*staticDirectoryNode)(nil)
var _ fs.NodeUnlinker = (*staticDirectoryNode)(nil)
var _ fs.NodeSymlinker = (*staticDirectoryNode)(nil)
var _ fs.NodeLinker = (*staticDirectoryNode)(nil)
var _ fs.NodeMknoder = (*staticDirectoryNode)(nil)
var _ fs.NodeSetattrer = (*staticDirectoryNode)(nil)
var _ fs.NodeGetattrer = (*staticDirectoryNode)(nil)
