// SPDX-License-Identifier: Apache-2.0

// +build !windows

package sysutil

import (
	"errors"
	"io"
	"os"
	"reflect"
	"testing"
)

func Test_userGetterImpl_getpwnam(t *testing.T) {
	tests := []struct {
		name        string
		passwdRaw   string
		readFileErr error
		username    string
		want        *User
		wantErr     error
	}{
		{
			"should return ErrGetUserFailed if failed to read passwd file",
			"",
			errors.New("read-error"),
			"root",
			nil,
			ErrGetUserFailed,
		},
		{
			"should skip invalid lines",
			`
invalid line 1
invalid line 2
sshd:x:105:65534::/run/sshd:/usr/sbin/nologin
invalid line 3
hlee:x:1000:1001::/home/hlee:/bin/bash`,
			nil,
			"hlee",
			&User{
				Name:    "hlee",
				UID:     1000,
				GID:     1001,
				HomeDir: "/home/hlee",
				Shell:   "/bin/bash",
			},
			nil,
		},
		{
			"should skip commented lines",
			`
#comment 1
# comment 2
sshd:x:105:65534::/run/sshd:/usr/sbin/nologin
# hlee:x:999:888::/home/hlee1:/bin/sh
hlee:x:1000:1001::/home/hlee:/bin/bash`,
			nil,
			"hlee",
			&User{
				Name:    "hlee",
				UID:     1000,
				GID:     1001,
				HomeDir: "/home/hlee",
				Shell:   "/bin/bash",
			},
			nil,
		},
		{
			"should return ErrGetUserFailed if user not found",
			`
root:x:0:0:root:/root:/bin/bash
daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin
bin:x:2:2:bin:/bin:/usr/sbin/nologin
sys:x:3:3:sys:/dev:/usr/sbin/nologin
sync:x:4:65534:sync:/bin:/bin/sync
games:x:5:60:games:/usr/games:/usr/sbin/nologin
man:x:6:12:man:/var/cache/man:/usr/sbin/nologin
lp:x:7:7:lp:/var/spool/lpd:/usr/sbin/nologin`,
			nil,
			"hlee",
			nil,
			ErrUserNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &osOperatorImpl{
				readFileFn: func(filename string) ([]byte, error) {
					return []byte(tt.passwdRaw), tt.readFileErr
				},
			}
			got, err := o.getpwnam(tt.username)
			if (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("getpwnam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getpwnam() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_osOperatorImpl_mkdir(t *testing.T) {
	type fields struct {
		osStatErr  error
		osMkDirErr error
		osChownErr error
	}

	dir := "path/to/dir"
	user := &User{
		Name:    "foo",
		UID:     1,
		GID:     2,
		HomeDir: "/home/path",
		Shell:   "/bin/sh",
	}
	perm := os.FileMode(0700)
	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			"should return nil if dir already exist",
			fields{osStatErr: nil},
			nil,
		},
		{
			"should return ErrMakeDirFailed if mkdir failed",
			fields{
				osStatErr:  os.ErrNotExist,
				osMkDirErr: errors.New("mkdir-err"),
				osChownErr: nil,
			},
			ErrMakeDirFailed,
		},
		{
			"should return ErrMakeDirFailed if chmod failed",
			fields{
				osStatErr:  os.ErrNotExist,
				osMkDirErr: nil,
				osChownErr: errors.New("chown-err"),
			},
			ErrMakeDirFailed,
		},
		{
			"should return ErrMakeDirFailed if stat returns other error",
			fields{
				osStatErr: errors.New("stat-err"),
			},
			ErrMakeDirFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &osOperatorImpl{
				osStatFn: func(name string) (os.FileInfo, error) {
					return nil, tt.fields.osStatErr
				},
				osMkDir: func(path string, perm os.FileMode) error {
					return tt.fields.osMkDirErr
				},
				osChown: func(name string, uid, gid int) error {
					return tt.fields.osChownErr
				},
			}
			if err := o.mkdir(dir, user, perm); (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("mkdir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type callRecorder struct {
	openFileCalled      bool
	openFileName        string
	openFileFlag        int
	openFilePerm        os.FileMode
	openFileErr         error
	openFileCloseCalled bool

	chownCalled bool
	chownFile   string
	chownUID    int
	chownGID    int
	chownErr    error

	removeCalled bool
	removeFile   string
}

func (c *callRecorder) Write(p []byte) (n int, err error) {
	//noop
	return 0, nil
}
func (c *callRecorder) Close() error {
	c.openFileCloseCalled = true
	return nil
}

func (c *callRecorder) openFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	c.openFileCalled = true
	c.openFileName = name
	c.openFileFlag = flag
	c.openFilePerm = perm
	return c, c.openFileErr
}

func (c *callRecorder) chown(name string, uid, gid int) error {
	c.chownCalled = true
	c.chownFile = name
	c.chownUID = uid
	c.chownGID = gid
	return c.chownErr
}
func (c *callRecorder) remove(name string) error {
	c.removeCalled = true
	c.removeFile = name
	return nil
}

func (c *callRecorder) dupReturnFields(c1 *callRecorder) {
	c.openFileErr = c1.openFileErr
	c.chownErr = c1.chownErr
}

func Test_osOperatorImpl_createFileForWrite(t *testing.T) {
	file := "path/to/new/file"
	user := &User{UID: 1, GID: 2}
	perm := os.FileMode(0655)

	type fields struct {
		openFileErr error
		chownErr    error
	}
	tests := []struct {
		name          string
		fields        fields
		expectedCalls *callRecorder
		wantErr       error
	}{
		{
			"should open file with os.O_WRONLY, os.O_CREATE and os.O_EXCL flag",
			fields{
				openFileErr: errors.New("open-file-err"),
			},
			&callRecorder{
				openFileCalled: true,
				openFileName:   file,
				openFileFlag:   os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
				openFilePerm:   perm,
			},
			ErrCreateFileFailed,
		},
		{
			"should remove file and return ErrCreateFileFailed if failed to chown",
			fields{
				chownErr: errors.New("chown-err"),
			},
			&callRecorder{
				openFileCalled:      true,
				openFileName:        file,
				openFileFlag:        os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
				openFilePerm:        perm,
				openFileCloseCalled: true,

				chownCalled: true,
				chownFile:   file,
				chownUID:    1,
				chownGID:    2,

				removeCalled: true,
				removeFile:   file,
			},
			ErrCreateFileFailed,
		},
		{
			"should work otherwise",
			fields{},
			&callRecorder{
				openFileCalled: true,
				openFileName:   file,
				openFileFlag:   os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
				openFilePerm:   perm,

				chownCalled: true,
				chownFile:   file,
				chownUID:    1,
				chownGID:    2,
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &callRecorder{
				openFileErr: tt.fields.openFileErr,
				chownErr:    tt.fields.chownErr,
			}
			o := &osOperatorImpl{
				osOpenFile: cr.openFile,
				osChown:    cr.chown,
				osRemove:   cr.remove,
			}
			_, err := o.createFileForWrite(file, user, perm)
			if (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("createFileForWrite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expectedCalls != nil {
				tt.expectedCalls.dupReturnFields(cr)
			}
			if !reflect.DeepEqual(cr, tt.expectedCalls) {
				t.Errorf("createFileForWrite() got = %v, want %v", cr, tt.expectedCalls)
			}
		})
	}
}
