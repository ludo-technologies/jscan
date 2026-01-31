package app

import (
	"os"
	"path/filepath"
	"strings"
)

// FileHelper provides file operation utilities
type FileHelper struct{}

// NewFileHelper creates a new FileHelper
func NewFileHelper() *FileHelper {
	return &FileHelper{}
}

// CollectJSFiles collects JavaScript/TypeScript files from paths
func (h *FileHelper) CollectJSFiles(paths []string, recursive bool, includePatterns, excludePatterns []string) ([]string, error) {
	var files []string

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			if h.isJSFile(path) && !h.isExcluded(path, excludePatterns) {
				files = append(files, path)
			}
			continue
		}

		// Directory handling
		if recursive {
			err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Skip excluded directories early
				if info.IsDir() {
					dirName := filepath.Base(filePath)
					for _, pattern := range excludePatterns {
						// Check for exact directory name match
						if pattern == dirName {
							return filepath.SkipDir
						}
						// Check for directory name with glob pattern
						if matched, _ := filepath.Match(pattern, dirName); matched {
							return filepath.SkipDir
						}
					}
					return nil
				}

				if h.isJSFile(filePath) && !h.isExcluded(filePath, excludePatterns) {
					files = append(files, filePath)
				}

				return nil
			})
		} else {
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					filePath := filepath.Join(path, entry.Name())
					if h.isJSFile(filePath) && !h.isExcluded(filePath, excludePatterns) {
						files = append(files, filePath)
					}
				}
			}
		}

		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// IsValidJSFile checks if a file is a valid JavaScript/TypeScript file
func (h *FileHelper) IsValidJSFile(path string) bool {
	return h.isJSFile(path)
}

// FileExists checks if a file exists
func (h *FileHelper) FileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

// ReadFile reads file content
func (h *FileHelper) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// isJSFile checks if a file is JavaScript/TypeScript based on extension
func (h *FileHelper) isJSFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".js" || ext == ".ts" || ext == ".jsx" || ext == ".tsx" ||
		ext == ".mjs" || ext == ".cjs" || ext == ".mts" || ext == ".cts"
}

// isExcluded checks if a path matches any exclude pattern
func (h *FileHelper) isExcluded(path string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		// Also check full path matching
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// ResolveFilePaths resolves file paths, returning existing files directly
// or collecting files from directories
func ResolveFilePaths(
	fileHelper *FileHelper,
	paths []string,
	recursive bool,
	includePatterns []string,
	excludePatterns []string,
) ([]string, error) {
	// Check if all paths are already files
	allFiles := true
	for _, path := range paths {
		exists, err := fileHelper.FileExists(path)
		if err != nil || !exists {
			allFiles = false
			break
		}
	}

	// If all paths are already files, no need to collect again
	if allFiles {
		return paths, nil
	}

	// Collect files from directories
	return fileHelper.CollectJSFiles(paths, recursive, includePatterns, excludePatterns)
}
