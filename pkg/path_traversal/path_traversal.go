package path_traversal

import (
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/dhenkes/gofman/pkg/gofman"
)

// Ensure service implements interface.
var _ gofman.PathTraversalService = (*PathTraversalService)(nil)

// PathTraversalService represents a service for looping through files and
// folders recursively.
type PathTraversalService struct{}

// NewPathTraversalService returns a new instance of PathTraversalService.
func NewPathTraversalService() *PathTraversalService {
	return &PathTraversalService{}
}

// Expand returns path using tilde expansion.
func (s *PathTraversalService) Expand(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
		return path, nil
	}

	u, err := user.Current()
	if err != nil {
		return path, err
	}

	if u.HomeDir == "" {
		return path, gofman.NewError(gofman.EINTERNAL, "Home directory not set.")
	}

	if path == "~" {
		return u.HomeDir, nil
	}

	fullpath := filepath.Join(u.HomeDir, strings.TrimPrefix(path, "~"+string(os.PathSeparator)))
	return fullpath, nil
}

// GetFilesInPath returns all files recursively starting from a root path.
func (s *PathTraversalService) GetFilesInPath(root string) ([]*gofman.File, error) {
	var files []*gofman.File

	err := filepath.WalkDir(root, func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dir.IsDir() {
			return nil
		}

		files = append(files, &gofman.File{
			Name: dir.Name(),
			Path: path,
		})

		return nil
	})

	return files, err
}
