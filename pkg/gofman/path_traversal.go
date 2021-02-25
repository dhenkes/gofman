package gofman

// PathTraversalService represents a service for looping through files and
// folders recursively.
type PathTraversalService interface {
	Expand(path string) (string, error)
	GetFilesInPath(root string) ([]*File, error)
}
