// SPDX-License-Identifier: Apache-2.0

//go:build !windows
// +build !windows

package sysutil

import (
	"errors"
	mock_os "github.com/digitalocean/droplet-agent/internal/sysutil/internal/mocks"
	"go.uber.org/mock/gomock"
	"os"
	"reflect"
	"testing"
)

func Test_osOperatorImpl_getpwnam(t *testing.T) {
	tests := []struct {
		name     string
		prepare  func(h *MockosOpHelper)
		username string
		want     *User
		wantErr  error
	}{
		{
			name: "should return ErrGetUserFailed if failed to read passwd file",
			prepare: func(h *MockosOpHelper) {
				h.EXPECT().ReadFile("/etc/passwd").Return(nil, errors.New("read-error"))
			},
			username: "root",
			want:     nil,
			wantErr:  ErrGetUserFailed,
		},
		{
			name: "should skip invalid lines",
			prepare: func(h *MockosOpHelper) {
				passwdRaw := `
invalid line 1
invalid line 2
sshd:x:105:65534::/run/sshd:/usr/sbin/nologin
invalid line 3
hlee:x:1000:1001::/home/hlee:/bin/bash`
				h.EXPECT().ReadFile("/etc/passwd").Return([]byte(passwdRaw), nil)
			},
			username: "hlee",
			want: &User{
				Name:    "hlee",
				UID:     1000,
				GID:     1001,
				HomeDir: "/home/hlee",
				Shell:   "/bin/bash",
			},
			wantErr: nil,
		},
		{
			name: "should skip commented lines",
			prepare: func(h *MockosOpHelper) {
				passwdRaw := `
#comment 1
# comment 2
sshd:x:105:65534::/run/sshd:/usr/sbin/nologin
# hlee:x:999:888::/home/hlee1:/bin/sh
hlee:x:1000:1001::/home/hlee:/bin/bash`
				h.EXPECT().ReadFile("/etc/passwd").Return([]byte(passwdRaw), nil)
			},
			username: "hlee",
			want: &User{
				Name:    "hlee",
				UID:     1000,
				GID:     1001,
				HomeDir: "/home/hlee",
				Shell:   "/bin/bash",
			},
			wantErr: nil,
		},
		{
			name: "should return ErrGetUserFailed if user not found",
			prepare: func(h *MockosOpHelper) {
				passwdRaw := `
root:x:0:0:root:/root:/bin/bash
daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin
bin:x:2:2:bin:/bin:/usr/sbin/nologin
sys:x:3:3:sys:/dev:/usr/sbin/nologin
sync:x:4:65534:sync:/bin:/bin/sync
games:x:5:60:games:/usr/games:/usr/sbin/nologin
man:x:6:12:man:/var/cache/man:/usr/sbin/nologin
lp:x:7:7:lp:/var/spool/lpd:/usr/sbin/nologin`
				h.EXPECT().ReadFile("/etc/passwd").Return([]byte(passwdRaw), nil)
			},
			username: "hlee",
			want:     nil,
			wantErr:  ErrUserNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			helperMock := NewMockosOpHelper(mockCtl)
			if tt.prepare != nil {
				tt.prepare(helperMock)
			}
			o := &osOperatorImpl{
				osOpHelper: helperMock,
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
		prepare func(h *MockosOpHelper)
		wantErr error
	}{
		{
			name: "return nil if dir already exist",
			prepare: func(h *MockosOpHelper) {
				h.EXPECT().Stat(dir).Return(&mock_os.MockFileInfo{}, nil)
			},
			wantErr: nil,
		},
		{
			name: "return ErrMakeDirFailed if mkdir failed",
			prepare: func(h *MockosOpHelper) {
				h.EXPECT().Stat(dir).Return(nil, os.ErrNotExist)
				h.EXPECT().MkDir(dir, perm).Return(errors.New("mkdir-err"))
			},
			wantErr: ErrMakeDirFailed,
		},
		{
			name: "return ErrMakeDirFailed if chmod failed",
			prepare: func(h *MockosOpHelper) {
				h.EXPECT().Stat(dir).Return(nil, os.ErrNotExist)
				h.EXPECT().MkDir(dir, perm).Return(nil)
				h.EXPECT().Chown(dir, user.UID, user.GID).Return(errors.New("chown-err"))
			},
			wantErr: ErrMakeDirFailed,
		},
		{
			name: "return ErrMakeDirFailed if stat returns other error",
			prepare: func(h *MockosOpHelper) {
				h.EXPECT().Stat(dir).Return(nil, errors.New("stat-err"))
			},
			wantErr: ErrMakeDirFailed,
		},
		{
			name: "happy path",
			prepare: func(h *MockosOpHelper) {
				h.EXPECT().Stat(dir).Return(nil, os.ErrNotExist)
				h.EXPECT().MkDir(dir, perm).Return(nil)
				h.EXPECT().Chown(dir, user.UID, user.GID).Return(nil)
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			helperMock := NewMockosOpHelper(mockCtl)
			if tt.prepare != nil {
				tt.prepare(helperMock)
			}
			o := &osOperatorImpl{
				osOpHelper: helperMock,
			}
			if err := o.mkdir(dir, user, perm); (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("mkdir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
