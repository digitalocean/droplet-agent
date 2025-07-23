// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/digitalocean/droplet-agent/internal/sysaccess/internal/mocks"
	"github.com/digitalocean/droplet-agent/internal/sysutil"

	"go.uber.org/mock/gomock"
)

func Test_Util(t *testing.T) {
	type args struct {
		home string   // temp directory for user home
		keys *os.File // temp auth keys file
		sys  *mocks.MocksysManager
	}

	getUser := func(a *args) *sysutil.User {
		return &sysutil.User{
			Name:    "test-user",
			UID:     1337,
			GID:     1337,
			HomeDir: a.home,
		}
	}

	tests := []struct {
		name    string
		expects func(*args) error
		stdin   string
		stdout  string
	}{
		{
			name: "with no arguments util will display the current users authorized keys",
			expects: func(a *args) error {
				user := getUser(a)
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(false, nil)
				a.sys.EXPECT().ReadFileOfUser(authKeys, user).Return([]byte("existing_key"), nil)

				return nil
			},
			stdout: "existing_key",
		},
		{
			name: "errors when reading the authorized keys are wrapped as ErrReadAuthorizedKeysFileFailed",
			expects: func(a *args) error {
				user := getUser(a)
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(false, nil)
				a.sys.EXPECT().ReadFileOfUser(authKeys, user).Return(nil, errors.New("some-read-error"))

				return ErrReadAuthorizedKeysFileFailed
			},
		},
		{
			name: "errors when determining the current user are wrapped as ErrReadAuthorizedKeysFileFailed",
			expects: func(a *args) error {
				a.sys.EXPECT().GetCurrentUser().Return(nil, errors.New("some-user-error"))
				return ErrReadAuthorizedKeysFileFailed
			},
		},
		{
			name: "errors when checking if the auth keys file is a symlink are wrapped as ErrReadAuthorizedKeysFileFailed",
			expects: func(a *args) error {
				user := getUser(a)
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(false, errors.New("some-symlink-error"))

				return ErrReadAuthorizedKeysFileFailed
			},
		},
		{
			name: "does not follow symlinks when reading the authorized keys file",
			expects: func(a *args) error {
				user := getUser(a)
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(true, nil)

				return ErrReadAuthorizedKeysFileFailed
			},
		},
		{
			name: "errors when checking if the file exists are wrapped as ErrReadAuthorizedKeysFileFailed",
			expects: func(a *args) error {
				user := getUser(a)
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, errors.New("some-exists-error"))

				return ErrReadAuthorizedKeysFileFailed
			},
		},
		{
			name: "if the file does not exist, no error is returned",
			expects: func(a *args) error {
				user := getUser(a)
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(false, nil)
				a.sys.EXPECT().ReadFileOfUser(authKeys, user).Return([]byte("some_key"), nil)

				return nil
			},
			stdout: "some_key",
		},
		{
			name: "does not follow symlinks when writing the authorized keys file",
			expects: func(a *args) error {
				user := getUser(a)
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(true, nil)

				return ErrWriteAuthorizedKeysFileFailed
			},
			stdin: "new_key",
		},
		{
			name: "when stdin is provided, will write the content to the users authorized keys file",
			expects: func(a *args) error {
				user := getUser(a)
				sshDir := path.Join(a.home, ".ssh")
				authKeys := path.Join(a.home, ".ssh", "authorized_keys")

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(false, nil)
				a.sys.EXPECT().MkDirIfNonExist(sshDir, user, os.FileMode(0700)).Return(nil)
				a.sys.EXPECT().CreateTempFile(sshDir, "authorized_keys-*.dotty", user).Return(a.keys, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(false, nil)
				a.sys.EXPECT().RenameFile(a.keys.Name(), authKeys).Return(nil)

				return nil
			},
			stdin: "new_key\nnew_key_2\nnew_key_3",
		},
		{
			name: "when an existing authorized keys file exists, attributes are copied",
			expects: func(a *args) error {
				user := getUser(a)
				sshDir := path.Join(a.home, ".ssh")
				authKeys := path.Join(a.home, ".ssh", "authorized_keys")

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(false, nil)
				a.sys.EXPECT().MkDirIfNonExist(sshDir, user, os.FileMode(0700)).Return(nil)
				a.sys.EXPECT().CreateTempFile(sshDir, "authorized_keys-*.dotty", user).Return(a.keys, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().CopyFileAttribute(authKeys, a.keys.Name()).Return(nil)
				a.sys.EXPECT().RenameFile(a.keys.Name(), authKeys).Return(nil)

				return nil
			},
			stdin: "new_key\nnew_key_2\nnew_key_3",
		},
		{
			name: "errors when determining the current user are wrapped as ErrWriteAuthorizedKeysFileFailed",
			expects: func(a *args) error {
				a.sys.EXPECT().GetCurrentUser().Return(nil, errors.New("some-user-error"))
				return ErrWriteAuthorizedKeysFileFailed
			},
			stdin: "new_key\nnew_key_2\nnew_key_3",
		},
		{
			name: "errors when checking if the auth keys file exists are wrapped as ErrWriteAuthorizedKeysFileFailed",
			expects: func(a *args) error {

				user := getUser(a)
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(false, errors.New("some-error"))

				return ErrWriteAuthorizedKeysFileFailed
			},
			stdin: "new_key\nnew_key_2\nnew_key_3",
		},
		{
			name: "when the auth keys file does not exist there is no symlink check",
			expects: func(a *args) error {
				user := getUser(a)
				sshDir := path.Join(a.home, ".ssh")
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(false, nil)
				a.sys.EXPECT().MkDirIfNonExist(sshDir, user, os.FileMode(0700)).Return(nil)
				a.sys.EXPECT().CreateTempFile(sshDir, "authorized_keys-*.dotty", user).Return(a.keys, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().CopyFileAttribute(authKeys, a.keys.Name()).Return(nil)
				a.sys.EXPECT().RenameFile(a.keys.Name(), authKeys).Return(nil)

				return nil
			},
			stdin: "new_key\nnew_key_2\nnew_key_3",
		},
		{
			name: "errors when checking if the auth keys file is a symlink are wrapped as ErrWriteAuthorizedKeysFileFailed",
			expects: func(a *args) error {
				user := getUser(a)
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(false, errors.New("some-error"))

				return ErrWriteAuthorizedKeysFileFailed
			},
			stdin: "new_key\nnew_key_2\nnew_key_3",
		},
		{
			name: "errors when creating the .ssh directory are wrapped as ErrWriteAuthorizedKeysFileFailed",
			expects: func(a *args) error {
				user := getUser(a)
				sshDir := path.Join(a.home, ".ssh")
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(false, nil)
				a.sys.EXPECT().MkDirIfNonExist(sshDir, user, os.FileMode(0700)).Return(errors.New("some-mkdir-error"))

				return ErrWriteAuthorizedKeysFileFailed
			},
			stdin: "new_key\nnew_key_2\nnew_key_3",
		},
		{
			name: "errors when creating the temp file are wrapped as ErrWriteAuthorizedKeysFileFailed",
			expects: func(a *args) error {
				user := getUser(a)
				sshDir := path.Join(a.home, ".ssh")
				authKeys := fmt.Sprintf("%s/.ssh/authorized_keys", a.home)

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(false, nil)
				a.sys.EXPECT().MkDirIfNonExist(sshDir, user, os.FileMode(0700)).Return(nil)
				a.sys.EXPECT().CreateTempFile(sshDir, "authorized_keys-*.dotty", user).Return(nil, errors.New("some-create-temp-error"))

				return ErrWriteAuthorizedKeysFileFailed
			},
			stdin: "new_key\nnew_key_2\nnew_key_3",
		},
		{
			name: "errors when checking if the auth keys file exists are wrapped as ErrWriteAuthorizedKeysFileFailed and the temp file is removed",
			expects: func(a *args) error {
				user := getUser(a)
				sshDir := path.Join(a.home, ".ssh")
				authKeys := path.Join(a.home, ".ssh", "authorized_keys")

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(false, nil)
				a.sys.EXPECT().MkDirIfNonExist(sshDir, user, os.FileMode(0700)).Return(nil)
				a.sys.EXPECT().CreateTempFile(sshDir, "authorized_keys-*.dotty", user).Return(a.keys, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(false, errors.New("some-file-exists-error"))
				a.sys.EXPECT().RemoveFile(a.keys.Name()).Return(nil)

				return ErrWriteAuthorizedKeysFileFailed
			},
			stdin: "new_key\nnew_key_2\nnew_key_3",
		},
		{
			name: "errors when copying file attributes are wrapped as ErrWriteAuthorizedKeysFileFailed and the temp file is removed, removal errors are ignored",
			expects: func(a *args) error {
				user := getUser(a)
				sshDir := path.Join(a.home, ".ssh")
				authKeys := path.Join(a.home, ".ssh", "authorized_keys")

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(false, nil)
				a.sys.EXPECT().MkDirIfNonExist(sshDir, user, os.FileMode(0700)).Return(nil)
				a.sys.EXPECT().CreateTempFile(sshDir, "authorized_keys-*.dotty", user).Return(a.keys, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().CopyFileAttribute(authKeys, a.keys.Name()).Return(errors.New("some-copy-error"))
				a.sys.EXPECT().RemoveFile(a.keys.Name()).Return(errors.New("some-remove-error"))

				return ErrWriteAuthorizedKeysFileFailed
			},
			stdin: "new_key\nnew_key_2\nnew_key_3",
		},
		{
			name: "errors when renaming the temp file to the auth keys file are wrapped as ErrWriteAuthorizedKeysFileFailed and the temp file is removed",
			expects: func(a *args) error {
				user := getUser(a)
				sshDir := path.Join(a.home, ".ssh")
				authKeys := path.Join(a.home, ".ssh", "authorized_keys")

				a.sys.EXPECT().GetCurrentUser().Return(user, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(true, nil)
				a.sys.EXPECT().IsSymLink(authKeys).Return(false, nil)
				a.sys.EXPECT().MkDirIfNonExist(sshDir, user, os.FileMode(0700)).Return(nil)
				a.sys.EXPECT().CreateTempFile(sshDir, "authorized_keys-*.dotty", user).Return(a.keys, nil)
				a.sys.EXPECT().FileExists(authKeys).Return(false, nil)
				a.sys.EXPECT().RenameFile(a.keys.Name(), authKeys).Return(errors.New("some-rename-error"))
				a.sys.EXPECT().RemoveFile(a.keys.Name()).Return(nil)

				return ErrWriteAuthorizedKeysFileFailed
			},
			stdin: "new_key\nnew_key_2\nnew_key_3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			tempDir, err := os.MkdirTemp("", "util_test_*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			tempFile, err := os.CreateTemp(tempDir, "authorized_keys-*.dotty")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer func() { _ = os.RemoveAll(tempDir) }()

			redirect, err := newRedirectIO(tt.stdin)
			if err != nil {
				t.Fatalf("failed to create redirect IO: %v", err)
			}

			sys := mocks.NewMocksysManager(mockCtl)
			mgr := NewUtilManager(sys)

			exErr := tt.expects(&args{
				home: tempDir,
				keys: tempFile,
				sys:  sys,
			})

			err = mgr.Util()
			if (exErr != nil && err == nil) || (err != nil && !errors.Is(err, exErr)) {
				t.Errorf("utilImpl.Util() error = %v, expects %v", err, exErr)
			}

			stdout, err := redirect.Capture()
			if err != nil {
				t.Fatalf("failed to capture stdout: %v", err)
			}

			if stdout != tt.stdout {
				t.Errorf("utilImpl.Util() output = %v, expects %v", stdout, tt.stdout)
			}

			if exErr == nil && tt.stdin != "" {
				// open the temp file in a new handle since it will be closed by now
				tempFile, err = os.Open(tempFile.Name())
				if err != nil {
					t.Fatalf("failed to reopen temp file: %v", err)
				}

				contents, err := io.ReadAll(tempFile)
				if err != nil {
					t.Fatalf("failed to read temp file: %v", err)
				}

				if keys := strings.TrimSpace(string(contents)); keys != tt.stdin {
					t.Errorf("utilImpl.Util() output = %v, expects %v", keys, tt.stdin)
				}
			}
		})
	}
}

// redirectIO is a utility to redirect stdin and stdout for testing purposes.
type redirectIO struct {
	ogStdout *os.File
	ogStdin  *os.File

	out chan []byte

	stdout *os.File
	stdin  *os.File
}

// newRedirectIO creates a new redirectIO instance to capture stdin and stdout.
func newRedirectIO(stdin string) (*redirectIO, error) {
	ogStdout := os.Stdout
	ogStdin := os.Stdin

	out := make(chan []byte)

	stdInR, stdInW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdOutR, stdOutW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	if _, err := stdInW.WriteString(stdin); err != nil {
		return nil, fmt.Errorf("failed to write to stdin pipe: %v", err)
	}

	_ = stdInW.Close()

	os.Stdin = stdInR
	os.Stdout = stdOutW

	go func() {
		buf := bytes.Buffer{}
		if _, err := io.Copy(&buf, stdOutR); err != nil {
			fmt.Printf("failed to read from stdout pipe: %v\n", err)
		}
		out <- buf.Bytes()
	}()

	return &redirectIO{
		ogStdout: ogStdout,
		ogStdin:  ogStdin,
		out:      out,
		stdout:   stdOutW,
		stdin:    stdInR,
	}, nil
}

// Capture returns the text sent to stdout since the creation of this redirectIO instance
func (r *redirectIO) Capture() (string, error) {
	if r.stdout == nil {
		return "", fmt.Errorf("stdout pipe is gone")
	}

	_ = os.Stdout.Close()
	out := <-r.out

	os.Stdout = r.ogStdout
	os.Stdin = r.ogStdin

	if r.stdout != nil {
		_ = r.stdout.Close()
		r.stdout = nil
	}

	if r.stdin != nil {
		_ = r.stdin.Close()
		r.stdin = nil
	}

	return strings.TrimSpace(string(out)), nil
}
