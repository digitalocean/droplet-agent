// SPDX-License-Identifier: Apache-2.0

package sysutil

import "os"

type osOpHelper interface {
	ReadFile(filename string) ([]byte, error)
	Stat(name string) (os.FileInfo, error)
	MkDir(path string, perm os.FileMode) error
	CreateTemp(dir, pattern string) (File, error)
	Chown(name string, uid, gid uint32) error
	Remove(name string) error
}

type osOpHelperImpl struct{}

func (*osOpHelperImpl) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename) //nolint:gosec
}

func (*osOpHelperImpl) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (*osOpHelperImpl) MkDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (*osOpHelperImpl) CreateTemp(dir, pattern string) (File, error) {
	return os.CreateTemp(dir, pattern)
}

func (*osOpHelperImpl) Chown(name string, uid, gid uint32) error {
	return os.Chown(name, int(uid), int(gid))
}

func (*osOpHelperImpl) Remove(name string) error {
	return os.Remove(name)
}
