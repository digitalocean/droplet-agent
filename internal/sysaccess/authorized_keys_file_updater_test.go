// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/sysaccess/internal/mocks"
	"github.com/digitalocean/droplet-agent/internal/sysutil"

	"go.uber.org/mock/gomock"
)

func Test_updaterImpl_updateAuthorizedKeysFile(t *testing.T) {
	log.Mute()

	authorizedKeyFileDir := "fixed/path/.ssh"
	authorizedKeyFile := authorizedKeyFileDir + "/authorized_keys"

	osUsername := "user1"

	getUserErr := errors.New("get-user-error")
	readKeyErr := errors.New("read-key-error")
	writeKeyErr := errors.New("write-key-error")

	validUser1 := &sysutil.User{
		Name:    osUsername,
		UID:     1,
		GID:     2,
		HomeDir: "/root",
	}

	validKey1 := &SSHKey{
		OSUser:     osUsername,
		PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
		ActorEmail: "actor1@email.com",
		TTL:        60,
	}

	tests := []struct {
		name    string
		prepare func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper)
		keys    []*SSHKey
		wantErr error
	}{
		{
			"should return error if failed to get os user",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper) {
				sysMgr.EXPECT().GetUserByName(osUsername).Return(nil, getUserErr)
			},
			[]*SSHKey{
				validKey1,
			},
			getUserErr,
		},
		{
			"should return ErrReadAuthorizedKeysFileFailed if failed to read existing file in subprocess",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper) {
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().UtilSubprocess(validUser1, nil).Return(nil, readKeyErr)
			},
			[]*SSHKey{
				validKey1,
			},
			ErrReadAuthorizedKeysFileFailed,
		},
		{
			"should return ErrReadAuthorizedKeysFileFailed if read subprocess returns non-zero exit code",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper) {
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().UtilSubprocess(validUser1, nil).Return(&sysutil.CmdResult{ExitCode: 1}, nil)
			},
			[]*SSHKey{
				validKey1,
			},
			ErrReadAuthorizedKeysFileFailed,
		},
		{
			"should return ErrWriteAuthorizedKeysFileFailed if failed to write keys in subprocess",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper) {
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().UtilSubprocess(validUser1, nil).Return(&sysutil.CmdResult{
					StdOut: "local-1\nlocal-2\n",
				}, nil).Times(1)
				sshHelper.EXPECT().prepareAuthorizedKeys([]string{"local-1", "local-2"}, []*SSHKey{validKey1}).Return([]string{"local-1", "local-2", "key"})
				sysMgr.EXPECT().UtilSubprocess(validUser1, []string{"local-1", "local-2", "key"}).Return(nil, writeKeyErr).Times(1)
			},
			[]*SSHKey{
				validKey1,
			},
			ErrWriteAuthorizedKeysFileFailed,
		},
		{
			"should return ErrWriteAuthorizedKeysFileFailed if write subprocess returns non-zero exit code",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper) {
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().UtilSubprocess(validUser1, nil).Return(nil, nil)
				sshHelper.EXPECT().prepareAuthorizedKeys(gomock.Any(), []*SSHKey{validKey1}).Return([]string{"key"})
				sysMgr.EXPECT().UtilSubprocess(validUser1, []string{"key"}).Return(&sysutil.CmdResult{ExitCode: 1}, nil)
			},
			[]*SSHKey{
				validKey1,
			},
			ErrWriteAuthorizedKeysFileFailed,
		},
		{
			"should proceed if authorized_keys does not exist",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper) {
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().UtilSubprocess(validUser1, nil).Return(nil, nil)
				sshHelper.EXPECT().prepareAuthorizedKeys(gomock.Any(), []*SSHKey{validKey1}).Return([]string{"key"})
				sysMgr.EXPECT().UtilSubprocess(validUser1, []string{"key"}).Return(nil, nil)
			},
			[]*SSHKey{
				validKey1,
			},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			sysMgrMock := mocks.NewMocksysManager(mockCtl)
			sshHelperMock := NewMocksshHelper(mockCtl)

			if tt.prepare != nil {
				tt.prepare(sysMgrMock, sshHelperMock)
			}

			sshMgr := &SSHManager{
				authorizedKeysFilePattern: authorizedKeyFile,
				sysMgr:                    sysMgrMock,
				sshHelper:                 sshHelperMock,
			}
			u := &updaterImpl{
				sshMgr: sshMgr,
			}

			if err := u.updateAuthorizedKeysFile(osUsername, tt.keys); !errors.Is(err, tt.wantErr) {
				t.Errorf("updateAuthorizedKeysFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_updaterImpl_updateAuthorizedKeysFile_threadSafe(t *testing.T) {
	t.Run("updateAuthorizedKeysFile must be thread safe", func(t *testing.T) {

		fakeKeys := []*SSHKey{{}}

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		sysMgrMock := mocks.NewMocksysManager(mockCtl)
		sshHelperMock := NewMocksshHelper(mockCtl)

		userNum := 5
		concurrentUpdatePerUser := 50

		runtime.GOMAXPROCS(userNum * concurrentUpdatePerUser)

		for i := 0; i != userNum; i++ {
			// set up expected calls for each user
			strUser := fmt.Sprintf("user_%d", i)
			user := &sysutil.User{
				Name:    strUser,
				UID:     uint32(i),
				GID:     1,
				HomeDir: fmt.Sprintf("/home/%s", strUser),
			}
			sysMgrMock.EXPECT().GetUserByName(strUser).Return(user, nil).Times(concurrentUpdatePerUser)

			keysFile := fmt.Sprintf("/home/%s/.ssh/authorized_keys", strUser)
			sshHelperMock.EXPECT().authorizedKeysFile(user).Return(keysFile).Times(concurrentUpdatePerUser)

			originalFile := ""
			for j := 0; j != concurrentUpdatePerUser; j++ {
				originalFile += "key\n"
				localKeys := strings.Split(strings.TrimRight(originalFile, "\n"), "\n")
				sysMgrMock.EXPECT().UtilSubprocess(user, nil).Return(&sysutil.CmdResult{
					StdOut: strings.Join(localKeys, "\n"),
				}, nil)
				allKeys := append(localKeys, "key")
				sshHelperMock.EXPECT().prepareAuthorizedKeys(localKeys, fakeKeys).Return(allKeys)
				sysMgrMock.EXPECT().UtilSubprocess(user, allKeys).Return(&sysutil.CmdResult{}, nil)
			}
		}

		sshMgr := &SSHManager{
			sysMgr:    sysMgrMock,
			sshHelper: sshHelperMock,
		}
		u := &updaterImpl{
			sshMgr: sshMgr,
		}

		var wg sync.WaitGroup
		var errs []error
		var errLock sync.Mutex
		for i := 0; i != userNum; i++ {
			strUser := fmt.Sprintf("user_%d", i)
			for j := 0; j != concurrentUpdatePerUser; j++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if e := u.updateAuthorizedKeysFile(strUser, fakeKeys); e != nil {
						errLock.Lock()
						errs = append(errs, e)
						errLock.Unlock()
					}
				}()
			}
		}
		wg.Wait()
		if len(errs) != 0 {
			t.Errorf("Unexpected Errors: %+v", errs)
		}
	})
}
