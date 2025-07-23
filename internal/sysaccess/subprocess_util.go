// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/sysutil"
)

// UtilManager can read or write the authorized_keys file
type UtilManager interface {
	// Util reads or writes the authorized_keys file based on stdin
	Util() error
}

type utilImpl struct {
	sys sysManager
}

// NewUtilManager returns a new UtilManager Object
func NewUtilManager(sys sysManager) UtilManager {
	if sys == nil {
		sys = sysutil.NewSysManager()
	}
	return &utilImpl{sys}
}

func (u *utilImpl) Util() error {
	stdin, err := readStdin(os.Stdin)
	if err != nil {
		return err
	}

	reading := stdin == ""
	wrapErr := func(err error, msg string) error {
		e := ErrWriteAuthorizedKeysFileFailed
		if reading {
			e = ErrReadAuthorizedKeysFileFailed
		}
		if msg == "" {
			return fmt.Errorf("%w: %v", e, err)
		}
		return fmt.Errorf("%w: %s: %v", e, msg, err)
	}

	usr, err := u.sys.GetCurrentUser()
	if err != nil {
		return wrapErr(err, "failed to get home dir")
	}

	authKeys, err := u.getAuthKeysPath(usr)
	if err != nil {
		return wrapErr(err, "invalid authorized_keys")
	}

	if reading {
		return u.read(usr, authKeys)
	}

	return u.write(usr, authKeys, stdin)
}

func (u *utilImpl) read(usr *sysutil.User, authKeys string) error {
	b, e := u.sys.ReadFileOfUser(authKeys, usr)
	if e != nil {
		if errors.Is(e, fs.ErrNotExist) || errors.Is(e, sysutil.ErrFileNotFound) {
			return nil
		}
		return fmt.Errorf("%w: %v", ErrReadAuthorizedKeysFileFailed, e)
	}

	content := strings.TrimSpace(string(b))
	if content != "" {
		fmt.Println(content)
	}

	return nil
}

func (u *utilImpl) write(usr *sysutil.User, authKeys string, stdin string) error {
	dir := path.Dir(authKeys)
	log.Debug("ensuring dir [%s] exists for user [%s]", dir, usr.Name)
	if err := u.sys.MkDirIfNonExist(dir, usr, 0700); err != nil {
		return fmt.Errorf("%w: failed to create directory %s: %v", ErrWriteAuthorizedKeysFileFailed, dir, err)
	}

	log.Debug("updating [%s]", authKeys)

	tmpFile, err := u.sys.CreateTempFile(dir, "authorized_keys-*.dotty", usr)
	if err != nil {
		return fmt.Errorf("%w: failed to create tmp file: %v", ErrWriteAuthorizedKeysFileFailed, err)
	}

	defer func() {
		log.Debug("[%s] updated", authKeys)
		_ = tmpFile.Close()
		if err != nil {
			_ = u.sys.RemoveFile(tmpFile.Name())
		}
	}()

	_, err = fmt.Fprintf(tmpFile, "%s\n", strings.TrimSpace(stdin))
	if err != nil {
		return fmt.Errorf("%w: failed to write to tmp file: %v", ErrWriteAuthorizedKeysFileFailed, err)
	}

	exists, err := u.sys.FileExists(authKeys)
	if err != nil {
		return fmt.Errorf("%w: failed to check if auth keys file exists: %v", ErrWriteAuthorizedKeysFileFailed, err)
	}

	if exists {
		log.Debug("copying file attribute from [%s] to [%s]", authKeys, tmpFile.Name())
		err = u.sys.CopyFileAttribute(authKeys, tmpFile.Name())
		if err != nil {
			return fmt.Errorf("%w: failed to apply file attribute: %v", ErrWriteAuthorizedKeysFileFailed, err)
		}
	}

	err = u.sys.RenameFile(tmpFile.Name(), authKeys)
	if err != nil {
		return fmt.Errorf("%w: failed to rename tmp file to auth keys file: %v", ErrWriteAuthorizedKeysFileFailed, err)
	}

	return nil
}

func (u *utilImpl) getAuthKeysPath(usr *sysutil.User) (string, error) {
	authKeys := path.Join(usr.HomeDir, ".ssh", "authorized_keys")

	exists, err := u.sys.FileExists(authKeys)
	if err != nil {
		return "", fmt.Errorf("failed to check if %s exists: %v", authKeys, err)
	}
	if !exists {
		return authKeys, nil
	}

	isSymLink, err := u.sys.IsSymLink(authKeys)
	if err != nil {
		return "", fmt.Errorf("failed to check if %s is a symlink: %v", authKeys, err)
	}
	if isSymLink {
		return "", fmt.Errorf("%s is a symlink, refusing to follow", authKeys)
	}

	return authKeys, nil
}

func readStdin(r io.Reader) (string, error) {
	if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", nil // No input on stdin
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return readStdinWithContext(ctx, r)
}

func readStdinWithContext(ctx context.Context, r io.Reader) (string, error) {
	var input []string
	scanner := bufio.NewScanner(r)

	done := make(chan struct{})
	var err error

	go func() {
		defer close(done)
		for scanner.Scan() {
			input = append(input, scanner.Text())
		}
		err = scanner.Err()
	}()

	select {
	case <-ctx.Done():
		return "", nil
	case <-done:
		if err != nil {
			return "", fmt.Errorf("error reading stdin: %w", err)
		}
		return strings.Join(input, "\n"), nil
	}
}
