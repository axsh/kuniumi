package kuniumi

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	vpath "path" // Renamed to avoid shadowing
	"path/filepath"
	"strings"
	"sync"
)

// VirtualEnvironment provides a sandboxed environment for functions.
// It abstracts file system operations and environment variable access, allowing functions
// to run consistently across different environments (local CLI, MCP, Container).
//
// It supports:
//   - **Environment Variables**: Managed set of environment variables separate from the host process.
//   - **File System Mounting**: Explicitly mounted directories map host paths to virtual paths.
//   - **Path Resolution**: Securely resolves virtual paths to host paths, preventing access outside mounted areas.
type VirtualEnvironment struct {
	envVars map[string]string
	fsRoot  string            // Root path for virtual FS (mapped from --mount)
	mounts  map[string]string // Host path -> Virtual path mapping

	pathMutex sync.RWMutex
	cwd       string // Virtual CWD, defaults to "/"
}

// envKey is the context key for VirtualEnvironment.
type envKey struct{}

// GetVirtualEnv retrieves the VirtualEnvironment from the context.
// If not found, it returns a safe, empty environment.
func GetVirtualEnv(ctx context.Context) *VirtualEnvironment {
	if v, ok := ctx.Value(envKey{}).(*VirtualEnvironment); ok {
		return v
	}
	return &VirtualEnvironment{
		envVars: make(map[string]string),
		cwd:     "/",
	}
}

// WithVirtualEnv adds the VirtualEnvironment to the context.
func WithVirtualEnv(ctx context.Context, v *VirtualEnvironment) context.Context {
	return context.WithValue(ctx, envKey{}, v)
}

// NewVirtualEnvironment creates a new environment.
func NewVirtualEnvironment(envVars map[string]string, mounts map[string]string) *VirtualEnvironment {
	if envVars == nil {
		envVars = make(map[string]string)
	}
	// Normalize mounts
	normalizedMounts := make(map[string]string)
	for h, v := range mounts {
		normalizedMounts[v] = h
	}

	return &VirtualEnvironment{
		envVars: envVars,
		mounts:  normalizedMounts,
		cwd:     "/",
	}
}

// --- Environment Variables ---

// Getenv retrieves the value of the environment variable named by the key.
// It looks up the variable in the virtual environment's managed set.
// If the variable is not present, it returns an empty string.
func (v *VirtualEnvironment) Getenv(key string) string {
	return v.envVars[key]
}

// ListEnv returns a copy of all environment variables in the virtual environment.
// The returned map is a copy, so modifying it does not affect the environment.
func (v *VirtualEnvironment) ListEnv() map[string]string {
	// Return a copy to prevent mutation
	c := make(map[string]string)
	for k, val := range v.envVars {
		c[k] = val
	}
	return c
}

// --- File System Operations ---

// resolvePath converts a virtual path to a real host path.
// It ensures the path is within the mounted directories.
func (v *VirtualEnvironment) resolvePath(virtualPath string) (string, error) {
	v.pathMutex.RLock()
	defer v.pathMutex.RUnlock()

	// Handle absolute/relative paths using forward slashes (virtual fs)
	p := virtualPath
	if !strings.HasPrefix(p, "/") {
		p = vpath.Join(v.cwd, p)
	}
	// Clean the path (resolve ..)
	p = vpath.Clean(p)

	// Logic: Find the longest matching mount prefix
	var bestMatchVirtual string
	var bestMatchHost string

	for virt, host := range v.mounts {
		// Ensure exact match or prefix
		if strings.HasPrefix(p, virt) {
			if len(virt) > len(bestMatchVirtual) {
				bestMatchVirtual = virt
				bestMatchHost = host
			}
		}
	}

	if bestMatchVirtual == "" {
		return "", fmt.Errorf("path not mounted: %s", virtualPath)
	}

	// Relativize path from the virtual mount point
	// Manual calculation since path package lacks Rel
	var rel string
	if p == bestMatchVirtual {
		rel = "."
	} else {
		// p starts with bestMatchVirtual
		rel = strings.TrimPrefix(p, bestMatchVirtual)
		rel = strings.TrimPrefix(rel, "/")
	}

	// Convert relative path to host OS separator
	relHost := filepath.FromSlash(rel)

	// Join with host path
	realPath := filepath.Join(bestMatchHost, relHost)

	return realPath, nil
}

// WriteFile writes data to a file at the specified virtual path.
// The file is created if it does not exist, or truncated if it does.
// The permission is fixed to 0644.
//
// Arguments:
//   - path: The virtual path to write to.
//   - data: The content to write.
//
// Returns an error if the path cannot be resolved (not mounted) or if the write fails.
func (v *VirtualEnvironment) WriteFile(path string, data []byte) error {
	realPath, err := v.resolvePath(path)
	if err != nil {
		return err
	}

	return os.WriteFile(realPath, data, 0644)
}

// ReadFile reads data from a file at the specified virtual path.
// It allows reading a specific chunk using offset and length.
//
// Arguments:
//   - path: The virtual path to read from.
//   - offset: The byte offset to start reading from.
//   - length: The number of bytes to read.
//
// Returns the read data, or an error if the path cannot be resolved or reading fails.
// If the file is shorter than (offset + length), it returns the available bytes.
func (v *VirtualEnvironment) ReadFile(path string, offset int64, length int64) ([]byte, error) {
	realPath, err := v.resolvePath(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(realPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if offset > 0 {
		_, err = f.Seek(offset, 0)
		if err != nil {
			return nil, err
		}
	}

	buf := make([]byte, length)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return buf[:n], nil
}

// RewriteFile overwrites data in a file at a specific offset.
// This preserves the rest of the file content.
//
// Arguments:
//   - path: The virtual path of the file.
//   - offset: The byte offset to start writing at.
//   - data: The new content to write.
//
// Returns an error if the file cannot be opened, path is invalid, or write fails.
func (v *VirtualEnvironment) RewriteFile(path string, offset int64, data []byte) error {
	realPath, err := v.resolvePath(path)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(realPath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteAt(data, offset)
	return err
}

// CopyFile copies a file from a source virtual path to a destination virtual path.
// It reads the entire source file and writes it to the destination.
//
// Arguments:
//   - src: The source virtual path.
//   - dst: The destination virtual path.
//
// Returns an error if either path cannot be resolved or IO operations fail.
func (v *VirtualEnvironment) CopyFile(src, dst string) error {
	realSrc, err := v.resolvePath(src)
	if err != nil {
		return err
	}
	realDst, err := v.resolvePath(dst)
	if err != nil {
		return err
	}

	input, err := os.ReadFile(realSrc)
	if err != nil {
		return err
	}

	return os.WriteFile(realDst, input, 0644)
}

// RemoveFile deletes the file at the specified virtual path.
//
// Warnings:
//   - This operation is permanent.
//   - It operates on the underlying host file system via the mount point.
func (v *VirtualEnvironment) RemoveFile(path string) error {
	realPath, err := v.resolvePath(path)
	if err != nil {
		return err
	}
	return os.Remove(realPath)
}

// Chmod changes the permissions of the file at the specified virtual path.
//
// Arguments:
//   - path: The virtual path of the file.
//   - mode: The new file mode (permissions).
func (v *VirtualEnvironment) Chmod(path string, mode os.FileMode) error {
	realPath, err := v.resolvePath(path)
	if err != nil {
		return err
	}
	return os.Chmod(realPath, mode)
}

// FileInfo is a simplified file info.
type FileInfo struct {
	Name  string
	Size  int64
	IsDir bool
}

// ListFile returns a list of files and directories in the specified virtual directory.
// It returns a slice of FileInfo structs containing name, size, and type information.
//
// Arguments:
//   - path: The virtual directory structure to list.
//
// Returns an error if the path is invalid or cannot be read.
func (v *VirtualEnvironment) ListFile(path string) ([]FileInfo, error) {
	realPath, err := v.resolvePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(realPath)
	if err != nil {
		return nil, err
	}

	var infos []FileInfo
	for _, e := range entries {
		info, _ := e.Info()
		infos = append(infos, FileInfo{
			Name:  e.Name(),
			Size:  info.Size(),
			IsDir: e.IsDir(),
		})
	}
	return infos, nil
}

// FindFile searches for files matching a pattern within a virtual directory.
//
// Arguments:
//   - root: The virtual root directory to start the search from.
//   - pattern: A shell pattern to match filenames (e.g. "*.go").
//   - recursive: If true, searches subdirectories recursively.
//
// Returns a list of matching virtual paths.
func (v *VirtualEnvironment) FindFile(root string, pattern string, recursive bool) ([]string, error) {
	realRoot, err := v.resolvePath(root)
	if err != nil {
		return nil, err
	}

	var matches []string

	// Walk function
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !recursive && path != realRoot && filepath.Dir(path) != realRoot {
			if d.IsDir() {
				return fs.SkipDir
			}
		}

		matched, err := filepath.Match(pattern, d.Name())
		if err != nil {
			return err
		}
		if matched {
			rel, _ := filepath.Rel(realRoot, path)
			// Convert host path separator to virtual (forward slash)
			relVirt := filepath.ToSlash(rel)
			virtualPath := vpath.Join(root, relVirt)
			matches = append(matches, virtualPath)
		}
		return nil
	}

	err = filepath.WalkDir(realRoot, walkFn)
	return matches, err
}

// ChangeCurrentDirectory changes the current working directory of the virtual environment.
// The new path must be a valid, existing directory within the mounted filesystems.
//
// Arguments:
//   - path: The target virtual path.
//
// Returns an error if the path does not exist, is not a directory, or cannot be resolved.
func (v *VirtualEnvironment) ChangeCurrentDirectory(path string) error {
	realPath, err := v.resolvePath(path)
	if err != nil {
		return err
	}

	stat, err := os.Stat(realPath)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}

	// Normalize virtual path
	p := path
	if !strings.HasPrefix(p, "/") {
		p = vpath.Join(v.cwd, p)
	}
	v.pathMutex.Lock()
	v.cwd = vpath.Clean(p)
	v.pathMutex.Unlock()

	return nil
}

// GetCurrentDirectory returns the current working directory of the virtual environment.
// The returned path is always a virtual path (e.g. "/src").
func (v *VirtualEnvironment) GetCurrentDirectory() string {
	v.pathMutex.RLock()
	defer v.pathMutex.RUnlock()
	return v.cwd
}
