// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This part copied directly from golang.org/src/path/filepath/path.go (Go
// 1.6) and lightly modified to be methods on BasicFilesystem.

// In our Walk() all paths given to a WalkFunc() are relative to the
// filesystem root.

package fs

import (
	"path/filepath"
	"sort"
)

// WalkFunc is the type of the function called for each directory
// visited by Walk. The path argument contains the argument to Walk as a
// prefix; that is, if Walk is called with "dir", which is a directory
// containing the file "a", the walk function will be called with argument
// "dir/a". The infos argument is the list of FileInfos in the named path.
//
// If there was a problem walking the directory named by path, the
// incoming error will describe the problem and the function can decide how
// to handle that error (and Walk will not descend into that directory). If
// an error is returned, processing stops. The sole exception is when the function
// returns the special value SkipDir. If the function returns SkipDir when invoked
// on a directory, Walk skips the directory's contents entirely.
// If the function returns SkipDir when invoked on a non-directory file,
// Walk skips the remaining files in the containing directory.
type WalkFunc func(path string, infos []FileInfo, err error) error

type walkFilesystem struct {
	Filesystem
}

func NewWalkFilesystem(next Filesystem) Filesystem {
	return &walkFilesystem{next}
}

// walk recursively descends path, calling walkFn.
func (f *walkFilesystem) walk(path string, walkFn WalkFunc) error {
	path, err := Canonicalize(path)
	if err != nil {
		return err
	}

	names, err := f.DirNames(path)
	if err != nil {
		return walkFn(path, nil, err)
	}

	sort.Strings(names)

	infos := make([]FileInfo, len(names))
	for i, name := range names {
		info, err := f.Lstat(filepath.Join(path, name))
		if err != nil {
			// ???
			continue
		}
		infos[i] = info
	}

	if err := walkFn(path, infos, nil); err != nil {
		return err
	}

	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		if err := f.walk(filepath.Join(path, info.Name()), walkFn); err != nil && err != SkipDir {
			return err
		}
	}

	return nil
}

// Walk walks the file tree rooted at root, calling walkFn for each
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The files are walked in lexical
// order, which makes the output deterministic but means that for very
// large directories Walk can be inefficient.
// Walk does not follow symbolic links.
func (f *walkFilesystem) Walk(root string, walkFn WalkFunc) error {
	return f.walk(root, walkFn)
}
