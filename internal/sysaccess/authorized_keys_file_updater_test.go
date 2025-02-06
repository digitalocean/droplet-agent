// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/sysaccess/internal/mocks"
	"github.com/digitalocean/droplet-agent/internal/sysutil"

	"go.uber.org/mock/gomock"
)

type recorder struct {
	bytes.Buffer
	closeCalled int
	expectedRes string
	filename    string
}

func (r *recorder) Close() error {
	r.closeCalled++
	return nil
}
func (r *recorder) Name() string {
	return r.filename
}
func (r *recorder) Stat() (os.FileInfo, error) {
	return nil, nil
}

func Test_updaterImpl_updateAuthorizedKeysFile(t *testing.T) {
	log.Mute()

	authorizedKeyFileDir := "fixed/path/.ssh"
	authorizedKeyFile := authorizedKeyFileDir + "/authorized_keys"

	osUsername := "user1"

	getUserErr := errors.New("get-user-error")
	mkDirErr := errors.New("make-dir-error")
	readFileErr := errors.New("read-file-error")
	createFileErr := errors.New("create-file-error")

	validUser1 := &sysutil.User{
		Name:    osUsername,
		UID:     1,
		GID:     2,
		HomeDir: "/root",
		Shell:   "/bin/bash",
	}

	validKey1 := &SSHKey{
		OSUser:     osUsername,
		PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
		ActorEmail: "actor1@email.com",
		TTL:        60,
	}

	tests := []struct {
		name       string
		prepare    func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper, recorder *recorder)
		keys       []*SSHKey
		wantErr    error
		wantRecord *recorder
	}{
		{
			"should return error if failed to get os user",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper, recorder *recorder) {
				sysMgr.EXPECT().GetUserByName(osUsername).Return(nil, getUserErr)
			},
			[]*SSHKey{
				validKey1,
			},
			getUserErr,
			nil,
		},
		{
			"should return error if failed to ensure authorized_keys dir exist",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper, recorder *recorder) {
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().MkDirIfNonExist(authorizedKeyFileDir, validUser1, os.FileMode(0700)).Return(mkDirErr)
			},
			[]*SSHKey{
				validKey1,
			},
			mkDirErr,
			nil,
		},
		{
			"should return ErrReadAuthorizedKeysFileFailed if failed to read existing file",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper, recorder *recorder) {
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().MkDirIfNonExist(authorizedKeyFileDir, validUser1, os.FileMode(0700)).Return(nil)
				sysMgr.EXPECT().ReadFileOfUser(authorizedKeyFile, validUser1).Return(nil, readFileErr)
			},
			[]*SSHKey{
				validKey1,
			},
			ErrReadAuthorizedKeysFileFailed,
			nil,
		},
		{
			"should proceed if authorized_keys not exist",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper, recorder *recorder) {
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().MkDirIfNonExist(authorizedKeyFileDir, validUser1, os.FileMode(0700)).Return(nil)
				sysMgr.EXPECT().ReadFileOfUser(authorizedKeyFile, validUser1).Return(nil, os.ErrNotExist)
				sshHelper.EXPECT().prepareAuthorizedKeys(gomock.Any(), gomock.Any()).Return([]string{})
				sysMgr.EXPECT().CreateTempFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, createFileErr)
			},
			[]*SSHKey{
				validKey1,
			},
			ErrWriteAuthorizedKeysFileFailed,
			nil,
		},
		{
			"should return ErrWriteAuthorizedKeysFileFailed if failed to create the file",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper, recorder *recorder) {
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().MkDirIfNonExist(authorizedKeyFileDir, validUser1, os.FileMode(0700)).Return(nil)
				sysMgr.EXPECT().ReadFileOfUser(authorizedKeyFile, validUser1).Return(nil, os.ErrNotExist)
				sshHelper.EXPECT().prepareAuthorizedKeys([]string{}, []*SSHKey{validKey1}).Return([]string{"line1", "line2"})
				sysMgr.EXPECT().CreateTempFile(authorizedKeyFileDir, "authorized_keys-*.dotty", validUser1).Return(nil, createFileErr)
			},
			[]*SSHKey{
				validKey1,
			},
			ErrWriteAuthorizedKeysFileFailed,
			nil,
		},
		{
			"should properly write files to tmp file and remove it if error happens",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper, recorder *recorder) {
				tmpFile := authorizedKeyFile + ".dotty"
				recorder.filename = tmpFile
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().MkDirIfNonExist(authorizedKeyFileDir, validUser1, os.FileMode(0700)).Return(nil)
				sysMgr.EXPECT().ReadFileOfUser(authorizedKeyFile, validUser1).Return([]byte{}, nil)
				sshHelper.EXPECT().prepareAuthorizedKeys([]string{""}, []*SSHKey{validKey1}).Return([]string{"line1", "line2"})
				sysMgr.EXPECT().CreateTempFile(authorizedKeyFileDir, "authorized_keys-*.dotty", validUser1).Return(recorder, nil)
				sysMgr.EXPECT().CopyFileAttribute(authorizedKeyFile, tmpFile).Return(nil)
				sysMgr.EXPECT().RenameFile(gomock.Any(), gomock.Any()).Return(errors.New("rename-error"))
				sysMgr.EXPECT().RemoveFile(tmpFile).Return(nil)
			},
			[]*SSHKey{
				validKey1,
			},
			ErrWriteAuthorizedKeysFileFailed,
			&recorder{
				closeCalled: 1,
				expectedRes: "line1\nline2\n",
			},
		},
		{
			"should properly write files to tmp file and rename it to original file",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper, recorder *recorder) {
				tmpFile := authorizedKeyFile + ".dotty"
				recorder.filename = tmpFile
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().MkDirIfNonExist(authorizedKeyFileDir, validUser1, os.FileMode(0700)).Return(nil)
				sysMgr.EXPECT().ReadFileOfUser(authorizedKeyFile, validUser1).Return([]byte{}, nil)
				sshHelper.EXPECT().prepareAuthorizedKeys([]string{""}, []*SSHKey{validKey1}).Return([]string{"line1", "line2"})
				sysMgr.EXPECT().CreateTempFile(authorizedKeyFileDir, "authorized_keys-*.dotty", validUser1).Return(recorder, nil)
				sysMgr.EXPECT().CopyFileAttribute(authorizedKeyFile, tmpFile).Return(nil)
				sysMgr.EXPECT().RenameFile(tmpFile, authorizedKeyFile).Return(nil)
			},
			[]*SSHKey{
				validKey1,
			},
			nil,
			&recorder{
				closeCalled: 1,
				expectedRes: "line1\nline2\n",
			},
		},
		{
			"should read existing keys and attempt to merge",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper, recorder *recorder) {
				tmpFile := authorizedKeyFile + ".dotty"
				recorder.filename = tmpFile
				localKeysRaw := []byte("local1\nlocal2\nlocal3\n\n\n")
				localKeys := []string{
					"local1", "local2", "local3",
				}
				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().MkDirIfNonExist(authorizedKeyFileDir, validUser1, os.FileMode(0700)).Return(nil)
				sysMgr.EXPECT().ReadFileOfUser(authorizedKeyFile, validUser1).Return(localKeysRaw, nil)
				sshHelper.EXPECT().prepareAuthorizedKeys(localKeys, []*SSHKey{validKey1}).Return([]string{"local1", "local2", "local3", "line1", "line2"})
				sysMgr.EXPECT().CreateTempFile(authorizedKeyFileDir, "authorized_keys-*.dotty", validUser1).Return(recorder, nil)
				sysMgr.EXPECT().CopyFileAttribute(authorizedKeyFile, tmpFile).Return(nil)
				sysMgr.EXPECT().RenameFile(tmpFile, authorizedKeyFile).Return(nil)
			},
			[]*SSHKey{
				validKey1,
			},
			nil,
			&recorder{
				closeCalled: 1,
				expectedRes: "local1\nlocal2\nlocal3\nline1\nline2\n",
			},
		},
		{
			"should skip copying file attribute if original file not exists",
			func(sysMgr *mocks.MocksysManager, sshHelper *MocksshHelper, recorder *recorder) {
				tmpFile := authorizedKeyFile + ".dotty"
				recorder.filename = tmpFile

				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().MkDirIfNonExist(authorizedKeyFileDir, validUser1, os.FileMode(0700)).Return(nil)
				sysMgr.EXPECT().ReadFileOfUser(authorizedKeyFile, validUser1).Return(nil, os.ErrNotExist)
				sshHelper.EXPECT().prepareAuthorizedKeys([]string{}, []*SSHKey{validKey1}).Return([]string{"line1", "line2"})
				sysMgr.EXPECT().CreateTempFile(authorizedKeyFileDir, "authorized_keys-*.dotty", validUser1).Return(recorder, nil)
				sysMgr.EXPECT().RenameFile(tmpFile, authorizedKeyFile).Return(nil)
			},
			[]*SSHKey{
				validKey1,
			},
			nil,
			&recorder{
				closeCalled: 1,
				expectedRes: "line1\nline2\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			sysMgrMock := mocks.NewMocksysManager(mockCtl)
			sshHelperMock := NewMocksshHelper(mockCtl)

			record := &recorder{}
			if tt.prepare != nil {
				tt.prepare(sysMgrMock, sshHelperMock, record)
			}

			sshMgr := &SSHManager{
				authorizedKeysFilePattern: authorizedKeyFile,
				sysMgr:                    sysMgrMock,
				sshHelper:                 sshHelperMock,
			}
			u := &updaterImpl{
				sshMgr: sshMgr,
			}
			if err := u.updateAuthorizedKeysFile(osUsername, tt.keys); (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("updateAuthorizedKeysFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantRecord != nil {
				if record.closeCalled != tt.wantRecord.closeCalled {
					t.Errorf("updateAuthorizedKeysFile() should properly close the file after each round")
				}
				if tt.wantRecord.expectedRes != record.String() {
					t.Errorf("updateAuthorizedKeysFile() should generate proper authorized_keys file content. want:\n %v\n\n got: \n%v", tt.wantRecord.expectedRes, record.String())
				}

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

		expectedRecords := make([][]string, userNum)
		recorders := make([][]*recorder, userNum)
		for i := range recorders {
			expectedRecords[i] = make([]string, concurrentUpdatePerUser)
			recorders[i] = make([]*recorder, concurrentUpdatePerUser)
			content := "key,"
			for j := range recorders[i] {
				recorders[i][j] = &recorder{}
				content += "key,"
				expectedRecords[i][j] = content
			}
		}

		for i := 0; i != userNum; i++ {
			// set up expected calls for each user
			strUser := fmt.Sprintf("user_%d", i)
			user := &sysutil.User{
				Name:    strUser,
				UID:     i,
				GID:     1,
				HomeDir: fmt.Sprintf("/home/%s", strUser),
				Shell:   "/bin/bash",
			}
			sysMgrMock.EXPECT().GetUserByName(strUser).Return(user, nil).Times(concurrentUpdatePerUser)

			keysFile := fmt.Sprintf("/home/%s/.ssh/authorized_keys", strUser)
			sshHelperMock.EXPECT().authorizedKeysFile(user).Return(keysFile).Times(concurrentUpdatePerUser)
			sysMgrMock.EXPECT().MkDirIfNonExist(filepath.Dir(keysFile), user, os.FileMode(0700)).Return(nil).Times(concurrentUpdatePerUser)

			tmpFilePath := keysFile + ".dotty"
			sysMgrMock.EXPECT().CopyFileAttribute(keysFile, tmpFilePath).Return(nil).Times(concurrentUpdatePerUser)
			sysMgrMock.EXPECT().RenameFile(tmpFilePath, keysFile).Return(nil).Times(concurrentUpdatePerUser)

			originalFile := ""
			for j := 0; j != concurrentUpdatePerUser; j++ {
				recorders[i][j].filename = tmpFilePath
				originalFile += "key\n"
				localKeys := strings.Split(strings.TrimRight(originalFile, "\n"), "\n")
				sysMgrMock.EXPECT().ReadFileOfUser(keysFile, user).Return([]byte(originalFile), nil)
				sshHelperMock.EXPECT().prepareAuthorizedKeys(localKeys, fakeKeys).Return(append(localKeys, "key"))
				sysMgrMock.EXPECT().CreateTempFile(filepath.Dir(keysFile), "authorized_keys-*.dotty", user).Return(recorders[i][j], nil)
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
		for i := range recorders {
			records := make([]string, 0, concurrentUpdatePerUser)
			for j := range recorders[i] {
				records = append(records, strings.ReplaceAll(recorders[i][j].String(), "\n", ","))
			}
			sort.Strings(records)
			if !reflect.DeepEqual(expectedRecords[i], records) {
				t.Errorf("user_%d, unexpected result!, want\n %+v \ngot\n %+v", i, expectedRecords[i], records)
			}
		}
	})
}
