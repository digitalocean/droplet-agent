// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/digitalocean/droplet-agent/internal/log"

	"github.com/digitalocean/droplet-agent/internal/sysutil"

	"github.com/golang/mock/gomock"

	"github.com/digitalocean/droplet-agent/internal/sysaccess/internal/mocks"
)

type recorder struct {
	bytes.Buffer
	closeCalled int
	expectedRes string
}

func (r *recorder) Close() error {
	r.closeCalled++
	return nil
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
				sysMgr.EXPECT().ReadFile(authorizedKeyFile).Return(nil, readFileErr)
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
				sysMgr.EXPECT().ReadFile(authorizedKeyFile).Return(nil, os.ErrNotExist)
				sshHelper.EXPECT().prepareAuthorizedKeys(gomock.Any(), gomock.Any()).Return([]string{})
				sysMgr.EXPECT().CreateFileForWrite(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, createFileErr)
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
				sysMgr.EXPECT().ReadFile(authorizedKeyFile).Return(nil, os.ErrNotExist)
				sshHelper.EXPECT().prepareAuthorizedKeys([]string{}, []*SSHKey{validKey1}).Return([]string{"line1", "line2"})
				sysMgr.EXPECT().CreateFileForWrite(authorizedKeyFile+".dotty", validUser1, os.FileMode(0600)).Return(nil, createFileErr)
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

				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().MkDirIfNonExist(authorizedKeyFileDir, validUser1, os.FileMode(0700)).Return(nil)
				sysMgr.EXPECT().ReadFile(authorizedKeyFile).Return(nil, os.ErrNotExist)
				sshHelper.EXPECT().prepareAuthorizedKeys([]string{}, []*SSHKey{validKey1}).Return([]string{"line1", "line2"})
				sysMgr.EXPECT().CreateFileForWrite(tmpFile, validUser1, os.FileMode(0600)).Return(recorder, nil)
				sysMgr.EXPECT().RunCmd("restorecon", tmpFile).Return(nil, nil)
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

				sysMgr.EXPECT().GetUserByName(osUsername).Return(validUser1, nil)
				sshHelper.EXPECT().authorizedKeysFile(validUser1).Return(authorizedKeyFile)
				sysMgr.EXPECT().MkDirIfNonExist(authorizedKeyFileDir, validUser1, os.FileMode(0700)).Return(nil)
				sysMgr.EXPECT().ReadFile(authorizedKeyFile).Return(nil, os.ErrNotExist)
				sshHelper.EXPECT().prepareAuthorizedKeys([]string{}, []*SSHKey{validKey1}).Return([]string{"line1", "line2"})
				sysMgr.EXPECT().CreateFileForWrite(tmpFile, validUser1, os.FileMode(0600)).Return(recorder, nil)
				sysMgr.EXPECT().RunCmd("restorecon", tmpFile).Return(nil, nil)
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
