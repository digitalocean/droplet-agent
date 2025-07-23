// SPDX-License-Identifier: Apache-2.0

package sysutil

import (
	"bytes"
	"errors"
	"os/user"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
)

func Test_GetCurrentUser(t *testing.T) {
	type args struct {
		userOp *MockuserOperator
	}

	tests := []struct {
		name    string
		expects func(*args) error
		asserts func(*testing.T, *User)
	}{
		{
			name: "happy path with valid user",
			expects: func(a *args) error {
				a.userOp.EXPECT().Current().Return(&user.User{
					Uid:      "1234",
					Gid:      "1234",
					Username: "some-user",
					Name:     "Some User",
					HomeDir:  "/home/some-user",
				}, nil)
				return nil
			},
			asserts: func(t *testing.T, u *User) {
				if u == nil {
					t.Fatal("expected user to be non-nil")
				}
				if u.UID != 1234 {
					t.Fatalf("expected UID to be 1234, got %d", u.UID)
				}
				if u.GID != 1234 {
					t.Fatalf("expected GID to be 1234, got %d", u.GID)
				}
				if u.Name != "some-user" {
					t.Fatalf("expected Name to be 'some-user', got '%s'", u.Name)
				}
				if u.HomeDir != "/home/some-user" {
					t.Fatalf("expected HomeDir to be '/home/some-user', got '%s'", u.HomeDir)
				}
			},
		},
		{
			name: "errors when looking up user are returned wrapped with ErrGetUserFailed",
			expects: func(a *args) error {
				a.userOp.EXPECT().Current().Return(nil, errors.New("user lookup error"))
				return ErrGetUserFailed
			},
		},
		{
			name: "invalid UIDs are wrapped with ErrGetUserFailed",
			expects: func(a *args) error {
				a.userOp.EXPECT().Current().Return(&user.User{
					Uid:      "potato",
					Gid:      "1234",
					Username: "some-user",
					Name:     "Some User",
					HomeDir:  "/home/some-user",
				}, nil)
				return ErrGetUserFailed
			},
		},
		{
			name: "invalid GIDs are wrapped with ErrGetUserFailed",
			expects: func(a *args) error {
				a.userOp.EXPECT().Current().Return(&user.User{
					Uid:      "1234",
					Gid:      "potato",
					Username: "some-user",
					Name:     "Some User",
					HomeDir:  "/home/some-user",
				}, nil)
				return ErrGetUserFailed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userOp := NewMockuserOperator(ctrl)

			mgr := NewSysManager()
			mgr.userOperator = userOp

			exErr := tt.expects(&args{userOp})

			u, err := mgr.GetCurrentUser()

			if !errors.Is(err, exErr) {
				t.Fatalf("expected error %v, got %v", exErr, err)
			}

			if tt.asserts != nil {
				tt.asserts(t, u)
			}
		})
	}
}

func Test_UtilSubprocess(t *testing.T) {
	type args struct {
		os   *MockosOperator
		user *User
		cmd  *Mockcmd
	}

	tests := []struct {
		name    string
		expects func(*args) error
		user    *User
		stdin   []string
		asserts func(*testing.T, *CmdResult)
	}{
		{
			name: "happy path with root user",
			expects: func(a *args) error {
				a.os.EXPECT().executable().Return("/path/to/bin/file", nil)
				a.os.EXPECT().evalSymLinks("/path/to/bin/file").Return("/long/path/to/bin/file", nil)
				a.os.EXPECT().command("/long/path/to/bin/file", "-util").Return(a.cmd)
				a.os.EXPECT().dir("/long/path/to/bin/file").Return("/long/path/to/bin")
				stdout := bytes.Buffer{}
				stderr := bytes.Buffer{}
				a.os.EXPECT().newBuffer().Return(stdout)
				a.os.EXPECT().newBuffer().Return(stderr)
				a.cmd.EXPECT().SetStdout(&stdout)
				a.cmd.EXPECT().SetStderr(&stderr)
				a.cmd.EXPECT().SetDir("/long/path/to/bin")
				a.cmd.EXPECT().SetUser(a.user)
				a.cmd.EXPECT().Run().Return(nil)
				return nil
			},
		},
		{
			name: "happy path with non-root user",
			expects: func(a *args) error {
				a.os.EXPECT().executable().Return("/path/to/bin/file", nil)
				a.os.EXPECT().evalSymLinks("/path/to/bin/file").Return("/long/path/to/bin/file", nil)
				a.os.EXPECT().command("/long/path/to/bin/file", "-util").Return(a.cmd)
				a.os.EXPECT().dir("/long/path/to/bin/file").Return("/long/path/to/bin")
				stdout := bytes.Buffer{}
				stderr := bytes.Buffer{}
				a.os.EXPECT().newBuffer().Return(stdout)
				a.os.EXPECT().newBuffer().Return(stderr)
				a.cmd.EXPECT().SetStdout(&stdout)
				a.cmd.EXPECT().SetStderr(&stderr)
				a.cmd.EXPECT().SetDir("/long/path/to/bin")
				a.cmd.EXPECT().SetUser(a.user)
				a.cmd.EXPECT().Run().Return(nil)
				return nil
			},
			user: &User{
				UID:     1000,
				GID:     1000,
				Name:    "some-user",
				HomeDir: "/home/some-user",
			},
		},
		{
			name: "stdin is set correctly",
			expects: func(a *args) error {
				a.os.EXPECT().executable().Return("/path/to/bin/file", nil)
				a.os.EXPECT().evalSymLinks("/path/to/bin/file").Return("/long/path/to/bin/file", nil)
				a.os.EXPECT().command("/long/path/to/bin/file", "-util").Return(a.cmd)
				a.os.EXPECT().dir("/long/path/to/bin/file").Return("/long/path/to/bin")
				stdout := bytes.Buffer{}
				stderr := bytes.Buffer{}
				stdin := strings.NewReader("key1\nkey2")
				a.os.EXPECT().newBuffer().Return(stdout)
				a.os.EXPECT().newBuffer().Return(stderr)
				a.os.EXPECT().newStringReader("key1\nkey2").Return(stdin)
				a.cmd.EXPECT().SetStdout(&stdout)
				a.cmd.EXPECT().SetStderr(&stderr)
				a.cmd.EXPECT().SetStdin(stdin)
				a.cmd.EXPECT().SetDir("/long/path/to/bin")
				a.cmd.EXPECT().SetUser(a.user)
				a.cmd.EXPECT().Run().Return(nil)
				return nil
			},
			stdin: []string{"key1", "key2"},
		},
		{
			name: "command errors are wrapped in ErrRunCmdFailed",
			expects: func(a *args) error {
				a.os.EXPECT().executable().Return("/path/to/bin/file", nil)
				a.os.EXPECT().evalSymLinks("/path/to/bin/file").Return("/long/path/to/bin/file", nil)
				a.os.EXPECT().command("/long/path/to/bin/file", "-util").Return(a.cmd)
				a.os.EXPECT().dir("/long/path/to/bin/file").Return("/long/path/to/bin")
				stdout := bytes.Buffer{}
				stderr := bytes.Buffer{}
				a.os.EXPECT().newBuffer().Return(stdout)
				a.os.EXPECT().newBuffer().Return(stderr)
				a.cmd.EXPECT().SetStdout(&stdout)
				a.cmd.EXPECT().SetStderr(&stderr)
				a.cmd.EXPECT().SetDir("/long/path/to/bin")
				a.cmd.EXPECT().SetUser(a.user)
				a.cmd.EXPECT().Run().Return(errors.New("some error"))
				return ErrRunCmdFailed
			},
			asserts: func(t *testing.T, res *CmdResult) {
				if res != nil {
					t.Fatal("expected CmdResult to be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockOs := NewMockosOperator(ctrl)

			mgr := NewSysManager()
			mgr.osOperator = mockOs

			mc := NewMockcmd(ctrl)

			usr := tt.user
			if usr == nil {
				usr = &User{
					UID:     0,
					GID:     0,
					Name:    "root",
					HomeDir: "/root",
				}
			}

			exErr := tt.expects(&args{
				mockOs,
				usr,
				mc,
			})

			res, err := mgr.UtilSubprocess(usr, tt.stdin)

			if !errors.Is(err, exErr) {
				t.Fatalf("expected error %v, got %v", exErr, err)
			}

			if tt.asserts != nil {
				tt.asserts(t, res)
			}
		})
	}
}
