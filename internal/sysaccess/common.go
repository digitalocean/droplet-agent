// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"errors"
	"os"
	"time"

	"github.com/digitalocean/droplet-agent/internal/sysutil"
)

// Possible errors
var (
	ErrSSHDConfigParseFailed         = errors.New("failed to parse sshd config")
	ErrInvalidKey                    = errors.New("invalid ssh key")
	ErrReadAuthorizedKeysFileFailed  = errors.New("failed to read authorized_keys file")
	ErrWriteAuthorizedKeysFileFailed = errors.New("failed to write authorized_keys file")
	ErrInvalidPortNumber             = errors.New("invalid port number")
	ErrInvalidArgs                   = errors.New("invalid arguments")
)

// SSHKeyType indicates the type of the ssh key.
// There are 2 types currently:
// - DOTTY: which is the keys used for web console sessions
// - Droplet: which is the droplet ssh keys managed through DigitalOcean
type SSHKeyType int

// constants for the SSH Key types
const (
	SSHKeyTypeDOTTY SSHKeyType = iota
	SSHKeyTypeDroplet
)

// SSHKey contains information of a ssh key operated by DOTTY
type SSHKey struct {
	OSUser     string `json:"os_user,omitempty"`
	PublicKey  string `json:"ssh_key"` // including algorithm and the key, separated by space (ASCII: 0x20)
	ActorEmail string `json:"actor_email"`
	TTL        int    `json:"ttl"` // time to live in seconds

	Type SSHKeyType `json:"-"` // key type

	fingerprint string
	expireAt    time.Time // set once when receiving the key, equals to receivedAt + TTL
}

type sshKeyInfo struct {
	OSUser     string `json:"os_user,omitempty"`
	ActorEmail string `json:"actor_email"`
	ExpireAt   string `json:"expire_at"`
}

type sysManager interface {
	GetUserByName(username string) (*sysutil.User, error)
	MkDirIfNonExist(dir string, user *sysutil.User, perm os.FileMode) error
	CreateTempFile(dir, pattern string, user *sysutil.User) (sysutil.File, error)
	CopyFileAttribute(from, to string) error
	ReadFile(filename string) ([]byte, error)
	ReadFileOfUser(filename string, user *sysutil.User) ([]byte, error)
	RenameFile(oldpath, newpath string) error
	RemoveFile(name string) error
	FileExists(name string) (bool, error)
	Sleep(d time.Duration)
}
