// Copyright (C) 2018 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package fs

import (
	"errors"
	"strings"
	"testing"
)

func TestWalk(t *testing.T) {
	// Walk should walk stuff in the expected order and pass the entries in
	// each directory to the walk function.

	fs := NewWalkFilesystem(newFakeFilesystem("/TestWalk"))

	// Some sample filesystem data

	fs.MkdirAll("a/aa", 0755)
	fs.MkdirAll("a/ab", 0755)
	fs.MkdirAll("a/ac", 0755)
	fs.MkdirAll("b/ba", 0755)
	fs.MkdirAll("b/bb", 0755)
	fs.Create("a/f0")
	fs.Create("a/f1")
	fs.Create("a/aa/f0")
	fs.Create("a/aa/f1")
	fs.Create("a/aa/F1")
	fs.Create("a/aa/f2")
	fs.Create("a/ab/f0")
	fs.Create("a/ab/f1")
	fs.Create("b/ba/f0")
	fs.Create("b/ba/f1")

	// The order and parameters to the walk function we expect

	results := []struct {
		dir   string
		names []string
	}{
		{".", []string{".stfolder", "a", "b"}},
		{".stfolder", []string{}},
		{"a", []string{"aa", "ab", "ac", "f0", "f1"}},
		{"a/aa", []string{"F1", "f0", "f1", "f2"}},
		{"a/ab", []string{"f0", "f1"}},
		{"a/ac", []string{}},
		{"b", []string{"ba", "bb"}},
		{"b/ba", []string{"f0", "f1"}},
		{"b/bb", []string{}},
	}

	// Walk and verify that the calls match the expectations

	i := 0
	walkFn := func(path string, infos []FileInfo, err error) error {
		if i >= len(results) {
			t.Fatal("Too many calls to the walk function")
		}
		exp := results[i]
		if path != exp.dir {
			t.Errorf("Call #%d for dir %q, expected %q", i, path, exp.dir)
		}
		if len(exp.names) != len(infos) {
			t.Errorf("Call #%d for dir %q got %d infos, expected %d", i, path, len(infos), len(exp.names))
		} else {
			for j := range infos {
				if infos[j].Name() != exp.names[j] {
					t.Errorf("Call #%d for dir %q name %d was %q, expected %q", i, path, j, infos[j].Name(), exp.names[j])
				}
			}
		}
		i++
		return nil
	}

	fs.Walk("/", walkFn)
	if i != len(results) {
		t.Fatalf("Got %d calls, expected %d", i, len(results))
	}
}

func TestWalkSkipDir(t *testing.T) {
	// Walk should prune the branch where SkipDir is returned but otherwise
	// continue walking.

	fs := NewWalkFilesystem(newFakeFilesystem("/TestWalkSkipDir"))

	// Some sample filesystem data

	fs.MkdirAll("a/aa/aaa/aaaa", 0755)
	fs.MkdirAll("a/aa/aab/aaba", 0755) // "aab" is the skip dir
	fs.MkdirAll("a/aa/aab/aabb", 0755) // these won't be walked
	fs.MkdirAll("a/aa/aac/aaca", 0755)
	fs.MkdirAll("a/ab/aba/abaa", 0755)

	// The order of directories called by the walk

	results := []string{
		".",
		".stfolder",
		"a",
		"a/aa",
		"a/aa/aaa",
		"a/aa/aaa/aaaa",
		"a/aa/aab", // here we return skipdir
		// "a/aa/aab/aaba", skipped
		// "a/aa/aab/aabb", skipped
		"a/aa/aac",
		"a/aa/aac/aaca",
		"a/ab",
		"a/ab/aba",
		"a/ab/aba/abaa",
	}

	// Walk and verify that the calls match the expectations

	i := 0
	walkFn := func(path string, infos []FileInfo, err error) error {
		if i >= len(results) {
			t.Fatal("Too many calls to the walk function")
		}
		if path != results[i] {
			t.Errorf("Call #%d for dir %q, expected %q", i, path, results[i])
		}

		i++

		if strings.HasSuffix(path, "/aab") {
			return SkipDir
		}
		return nil
	}

	fs.Walk("/", walkFn)
	if i != len(results) {
		t.Fatalf("Got %d calls, expected %d", i, len(results))
	}
}

func TestWalkOtherError(t *testing.T) {
	// Walk should stop walking completely when a non-SkipDir error is
	// returned.

	fs := NewWalkFilesystem(newFakeFilesystem("/TestWalkOtherError"))

	// Some sample filesystem data

	fs.MkdirAll("a/aa/aaa/aaaa", 0755)
	fs.MkdirAll("a/aa/aab/aaba", 0755) // "aab" is the skip dir
	fs.MkdirAll("a/aa/aab/aabb", 0755) // the rest should not be walked
	fs.MkdirAll("a/aa/aac/aaca", 0755)
	fs.MkdirAll("a/ab/aba/abaa", 0755)

	// The order of directories called by the walk

	results := []string{
		".",
		".stfolder",
		"a",
		"a/aa",
		"a/aa/aaa",
		"a/aa/aaa/aaaa",
		"a/aa/aab", // here we return error
		// nothing more happens
	}

	// Walk and verify that the calls match the expectations

	i := 0
	walkFn := func(path string, infos []FileInfo, err error) error {
		if i >= len(results) {
			t.Fatal("Too many calls to the walk function")
		}
		if path != results[i] {
			t.Errorf("Call #%d for dir %q, expected %q", i, path, results[i])
		}

		i++

		if strings.HasSuffix(path, "/aab") {
			return errors.New("grr argh")
		}
		return nil
	}

	fs.Walk("/", walkFn)
	if i != len(results) {
		t.Fatalf("Got %d calls, expected %d", i, len(results))
	}
}
