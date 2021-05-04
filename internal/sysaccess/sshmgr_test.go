package sysaccess

import (
	"errors"
	"reflect"
	"testing"

	"github.com/digitalocean/droplet-agent/internal/log"

	"github.com/digitalocean/droplet-agent/internal/sysaccess/internal/mocks"

	"github.com/golang/mock/gomock"
)

func TestSSHManager_parseSSHDConfig(t *testing.T) {
	tests := []struct {
		name                   string
		sshdCfg                string
		sshdCfgReadErr         error
		wantAuthorizedKeysFile string
		wantErr                error
	}{
		{
			"should return ErrSSHDConfigParseFailed if failed to read sshd_config file",
			"",
			errors.New("read-err"),
			defaultAuthorizedKeysFile,
			ErrSSHDConfigParseFailed,
		},
		{
			"should skip unrelated lines",
			"\t# unrelated line 1  \n# AuthorizedKeysFile /wrong/key/file\n\t   AuthorizedKeysFile /etc/ssh/sshd.conf/%u    ",
			nil,
			"/etc/ssh/sshd.conf/%u",
			nil,
		},
		{
			"should correctly support the case of consecutive spaces",
			"\t   \tAuthorizedKeysFile     /etc/ssh/sshd.conf/%u",
			nil,
			"/etc/ssh/sshd.conf/%u",
			nil,
		},
		{
			"should correctly support \\t separator",
			"\t   \tAuthorizedKeysFile \t\t/etc/ssh/sshd.conf/%u",
			nil,
			"/etc/ssh/sshd.conf/%u",
			nil,
		},
		{
			"should only fetch the first pattern",
			"AuthorizedKeysFile /etc/ssh/sshd.conf/%u %h/second/ssh/keys",
			nil,
			"/etc/ssh/sshd.conf/%u",
			nil,
		},
		{
			"should translate relative path",
			"AuthorizedKeysFile .ssh/authorized_keys",
			nil,
			"%h/.ssh/authorized_keys",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			sysMgrMock := mocks.NewMocksysManager(mockCtl)
			sysMgrMock.EXPECT().ReadFile(gomock.Any()).Return([]byte(tt.sshdCfg), tt.sshdCfgReadErr)
			s := &SSHManager{
				sysMgr: sysMgrMock,
			}
			s.sshHelper = &sshHelperImpl{mgr: s}

			if err := s.parseSSHDConfig(); (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("parseSSHDConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if s.authorizedKeysFilePattern != tt.wantAuthorizedKeysFile {
				t.Errorf("parseSSHDConfig() AuthorizedKeysFile got = %v, want %v", s.authorizedKeysFilePattern, tt.wantAuthorizedKeysFile)
			}
		})
	}
}

func TestSSHManager_UpdateKeys(t *testing.T) {
	log.Mute()

	username1 := "user1"
	key1 := &SSHKey{
		OSUser:    username1,
		PublicKey: "public-key-1",
		TTL:       123,
	}
	key11 := &SSHKey{
		OSUser:    username1,
		PublicKey: "public-key-11",
		TTL:       123,
	}

	username2 := "user2"
	key21 := &SSHKey{
		OSUser:    username2,
		PublicKey: "public-key-21",
		TTL:       123,
	}
	key22 := &SSHKey{
		OSUser:    username2,
		PublicKey: "public-key-22",
		TTL:       123,
	}

	invalidKeyErr := errors.New("invalid-key")
	failedUpdateErr := errors.New("failed-update")

	tests := []struct {
		name           string
		prepare        func(sshMgr *SSHManager, sshHpr *MocksshHelper, updater *MockauthorizedKeysFileUpdater)
		keys           []*SSHKey
		wantErr        error
		wantCachedKeys map[string][]*SSHKey
	}{
		{
			"should removed expired keys from the cached keys before proceeding and return error when any of the key is invalid",
			func(sshMgr *SSHManager, sshHpr *MocksshHelper, updater *MockauthorizedKeysFileUpdater) {
				sshHpr.EXPECT().validateKey(key1).Return(invalidKeyErr)
			},
			[]*SSHKey{key1},
			invalidKeyErr,
			nil,
		},
		{
			"should group the keys by user and do not update keys for a user if unchanged",
			func(sshMgr *SSHManager, sshHpr *MocksshHelper, updater *MockauthorizedKeysFileUpdater) {
				sshMgr.cachedKeys = map[string][]*SSHKey{
					username1: {key1},
					username2: {key21, key22},
				}
				sshHpr.EXPECT().validateKey(gomock.Any()).Return(nil).Times(3)
				sshHpr.EXPECT().areSameKeys([]*SSHKey{key11}, sshMgr.cachedKeys[username1]).
					Return(false)
				updater.EXPECT().updateAuthorizedKeysFile(username1, []*SSHKey{key11}).Return(nil)
				sshHpr.EXPECT().areSameKeys([]*SSHKey{key21, key22}, sshMgr.cachedKeys[username2]).
					Return(true)
			},
			[]*SSHKey{key11, key21, key22},
			nil,
			map[string][]*SSHKey{
				username1: {key11},
				username2: {key21, key22},
			},
		},
		{
			"should return error if failed to update key and do not update cached keys",
			func(sshMgr *SSHManager, sshHpr *MocksshHelper, updater *MockauthorizedKeysFileUpdater) {
				sshMgr.cachedKeys = map[string][]*SSHKey{
					username1: {key1},
				}
				sshHpr.EXPECT().validateKey(gomock.Any()).Return(nil)
				sshHpr.EXPECT().areSameKeys([]*SSHKey{key11}, sshMgr.cachedKeys[username1]).
					Return(false)
				updater.EXPECT().updateAuthorizedKeysFile(username1, []*SSHKey{key11}).Return(failedUpdateErr)
			},
			[]*SSHKey{key11},
			failedUpdateErr,
			map[string][]*SSHKey{
				username1: {key1},
			},
		},
		{
			"should work if metadata returned keys for a new user",
			func(sshMgr *SSHManager, sshHpr *MocksshHelper, updater *MockauthorizedKeysFileUpdater) {
				sshMgr.cachedKeys = map[string][]*SSHKey{
					username1: {key1},
				}
				sshHpr.EXPECT().validateKey(gomock.Any()).Return(nil).Times(3)

				sshHpr.EXPECT().areSameKeys([]*SSHKey{key1}, sshMgr.cachedKeys[username1]).
					Return(true)

				sshHpr.EXPECT().areSameKeys([]*SSHKey{key21, key22}, nil).
					Return(false)
				updater.EXPECT().updateAuthorizedKeysFile(username2, []*SSHKey{key21, key22}).Return(nil)
			},
			[]*SSHKey{key1, key21, key22},
			nil,
			map[string][]*SSHKey{
				username1: {key1},
				username2: {key21, key22},
			},
		},
		{
			"should work if metadata removed keys for an existing user",
			func(sshMgr *SSHManager, sshHpr *MocksshHelper, updater *MockauthorizedKeysFileUpdater) {
				sshMgr.cachedKeys = map[string][]*SSHKey{
					username1: {key1},
					username2: {key21, key22},
				}
				sshHpr.EXPECT().validateKey(gomock.Any()).Return(nil)
				sshHpr.EXPECT().areSameKeys([]*SSHKey{key1}, []*SSHKey{key1}).
					Return(true)

				updater.EXPECT().updateAuthorizedKeysFile(username2, nil).Return(nil)
			},
			[]*SSHKey{key1},
			nil,
			map[string][]*SSHKey{
				username1: {key1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			sshHelperMock := NewMocksshHelper(mockCtl)
			updaterMock := NewMockauthorizedKeysFileUpdater(mockCtl)

			s := &SSHManager{
				sshHelper:                 sshHelperMock,
				authorizedKeysFileUpdater: updaterMock,
			}
			if tt.prepare != nil {
				tt.prepare(s, sshHelperMock, updaterMock)
			}
			if err := s.UpdateKeys(tt.keys); (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("UpdateKeys() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantCachedKeys != nil && !reflect.DeepEqual(tt.wantCachedKeys, s.cachedKeys) {
				t.Errorf("UpdateKeys() didn't update the cached keys,got = %v, want %v", s.cachedKeys, tt.wantCachedKeys)
			}
		})
	}
	// TODO: add another test for applying lock, maybe?
}

func TestSSHManager_RemoveExpiredKeys(t *testing.T) {
	log.Mute()

	username1 := "user1"
	key1 := &SSHKey{
		OSUser:    username1,
		PublicKey: "public-key-1",
		TTL:       123,
	}
	key11 := &SSHKey{
		OSUser:    username1,
		PublicKey: "public-key-11",
		TTL:       123,
	}

	username2 := "user2"
	key21 := &SSHKey{
		OSUser:    username2,
		PublicKey: "public-key-21",
		TTL:       123,
	}
	key22 := &SSHKey{
		OSUser:    username2,
		PublicKey: "public-key-22",
		TTL:       123,
	}

	failedUpdateErr := errors.New("failed-update")

	tests := []struct {
		name           string
		prepare        func(sshMgr *SSHManager, sshHpr *MocksshHelper, updater *MockauthorizedKeysFileUpdater)
		wantErr        error
		wantCachedKeys map[string][]*SSHKey
	}{
		{
			"should return nil if no key cached",
			func(sshMgr *SSHManager, sshHpr *MocksshHelper, updater *MockauthorizedKeysFileUpdater) {
				sshMgr.cachedKeys = map[string][]*SSHKey{}
			},
			nil,
			nil,
		},
		{
			"should not update if all keys still valid",
			func(sshMgr *SSHManager, sshHpr *MocksshHelper, updater *MockauthorizedKeysFileUpdater) {
				sshMgr.cachedKeys = map[string][]*SSHKey{
					username1: {key1},
				}
				sshHpr.EXPECT().removeExpiredKeys(sshMgr.cachedKeys).Return(sshMgr.cachedKeys)
				sshHpr.EXPECT().areSameKeys([]*SSHKey{key1}, sshMgr.cachedKeys[username1]).
					Return(true)
			},
			nil,
			map[string][]*SSHKey{
				username1: {key1},
			},
		},
		{
			"should return error if failed to update key and do not update the cached keys",
			func(sshMgr *SSHManager, sshHpr *MocksshHelper, updater *MockauthorizedKeysFileUpdater) {
				sshMgr.cachedKeys = map[string][]*SSHKey{
					username1: {key1, key11},
				}
				sshHpr.EXPECT().removeExpiredKeys(sshMgr.cachedKeys).
					Return(map[string][]*SSHKey{
						username1: {key1},
					})
				sshHpr.EXPECT().areSameKeys([]*SSHKey{key1, key11}, []*SSHKey{key1}).
					Return(false)
				updater.EXPECT().updateAuthorizedKeysFile(username1, []*SSHKey{key1}).Return(failedUpdateErr)
			},
			failedUpdateErr,
			map[string][]*SSHKey{
				username1: {key1, key11},
			},
		},
		{
			"should work if all keys for a user expired",
			func(sshMgr *SSHManager, sshHpr *MocksshHelper, updater *MockauthorizedKeysFileUpdater) {
				sshMgr.cachedKeys = map[string][]*SSHKey{
					username1: {key1, key11},
					username2: {key21, key22},
				}
				sshHpr.EXPECT().removeExpiredKeys(sshMgr.cachedKeys).
					Return(map[string][]*SSHKey{
						username1: {key1},
					})
				sshHpr.EXPECT().areSameKeys([]*SSHKey{key1, key11}, []*SSHKey{key1}).
					Return(false)
				updater.EXPECT().updateAuthorizedKeysFile(username1, []*SSHKey{key1}).Return(nil)

				sshHpr.EXPECT().areSameKeys([]*SSHKey{key21, key22}, nil).
					Return(false)
				updater.EXPECT().updateAuthorizedKeysFile(username2, nil).Return(nil)
			},
			nil,
			map[string][]*SSHKey{
				username1: {key1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			sshHelperMock := NewMocksshHelper(mockCtl)
			updaterMock := NewMockauthorizedKeysFileUpdater(mockCtl)

			s := &SSHManager{
				sshHelper:                 sshHelperMock,
				authorizedKeysFileUpdater: updaterMock,
			}
			if tt.prepare != nil {
				tt.prepare(s, sshHelperMock, updaterMock)
			}
			if err := s.RemoveExpiredKeys(); (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("RemoveExpiredKeys() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantCachedKeys != nil && !reflect.DeepEqual(tt.wantCachedKeys, s.cachedKeys) {
				t.Errorf("RemoveExpiredKeys() didn't update the cached keys,got = %v, want %v", s.cachedKeys, tt.wantCachedKeys)
			}
		})
	}
}
