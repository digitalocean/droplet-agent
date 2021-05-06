package sysaccess

import (
	"errors"
	"io"
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
)

// SSHKey contains information of a ssh key operated by DOTTY
type SSHKey struct {
	OSUser     string `json:"os_user,omitempty"`
	PublicKey  string `json:"ssh_key"` // including algorithm and the key, separated by space (ASCII: 0x20)
	ActorEmail string `json:"actor_email"`
	TTL        int    `json:"ttl"` // time to live in seconds

	expireAt time.Time // set once when receiving the key, equals to receivedAt + TTL
}

type sshKeyInfo struct {
	OSUser     string `json:"os_user,omitempty"`
	ActorEmail string `json:"actor_email"`
	ExpireAt   string `json:"expire_at"`
}

type sysManager interface {
	GetUserByName(username string) (*sysutil.User, error)
	MkDirIfNonExist(dir string, user *sysutil.User, perm os.FileMode) error
	CreateFileIfNonExist(file string, user *sysutil.User, perm os.FileMode) (io.WriteCloser, error)
	RunCmd(name string, arg ...string) (*sysutil.CmdResult, error)
	ReadFile(filename string) ([]byte, error)
	RenameFile(oldpath, newpath string) error
	RemoveFile(name string) error
}
