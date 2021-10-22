// SPDX-License-Identifier: Apache-2.0

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
	log.Mute()
	tests := []struct {
		name                   string
		prepare                func(s *SSHManager)
		sshdCfg                string
		sshdCfgReadErr         error
		wantAuthorizedKeysFile string
		wantSSHDPort           int
		wantErr                error
	}{
		{
			"should return ErrSSHDConfigParseFailed if failed to read sshd_config file",
			nil,
			"",
			errors.New("read-err"),
			defaultAuthorizedKeysFile,
			defaultSSHDPort,
			ErrSSHDConfigParseFailed,
		},
		{
			"should use default values if not configured in sshd_config",
			nil,
			"\t# useless line 1 \n \t\t\t # non related config 123",
			nil,
			defaultAuthorizedKeysFile,
			defaultSSHDPort,
			nil,
		},
		{
			"should skip unrelated lines",
			nil,
			"\t# unrelated line 1  \n# AuthorizedKeysFile /wrong/key/file\n\t   AuthorizedKeysFile /etc/ssh/sshd.conf/%u    \nPort 114",
			nil,
			"/etc/ssh/sshd.conf/%u",
			114,
			nil,
		},
		{
			"should correctly support the case of consecutive spaces",
			nil,
			"\t   \tAuthorizedKeysFile     /etc/ssh/sshd.conf/%u",
			nil,
			"/etc/ssh/sshd.conf/%u",
			defaultSSHDPort,
			nil,
		},
		{
			"invalid config result in default AuthorizedKeysFile pattern",
			nil,
			"AuthorizedKeysFile # /etc/ssh/sshd.conf/%u",
			nil,
			defaultAuthorizedKeysFile,
			defaultSSHDPort,
			nil,
		},
		{
			"should correctly support \\t separator",
			nil,
			"\t   \tAuthorizedKeysFile\t\t/etc/ssh/sshd.conf/%u",
			nil,
			"/etc/ssh/sshd.conf/%u",
			defaultSSHDPort,
			nil,
		},
		{
			"should only fetch the first pattern",
			nil,
			"AuthorizedKeysFile /etc/ssh/sshd.conf/%u %h/second/ssh/keys",
			nil,
			"/etc/ssh/sshd.conf/%u",
			defaultSSHDPort,
			nil,
		},
		{
			"should translate relative path",
			nil,
			"AuthorizedKeysFile .ssh/authorized_keys",
			nil,
			"%h/.ssh/authorized_keys",
			defaultSSHDPort,
			nil,
		},
		{
			"should ignore comment",
			nil,
			"AuthorizedKeysFile /etc/ssh/sshd.conf/%u\t# this is a comment",
			nil,
			"/etc/ssh/sshd.conf/%u",
			defaultSSHDPort,
			nil,
		},
		{
			"comment can start right after the AuthorizedKeysFile config without a separator",
			nil,
			"AuthorizedKeysFile /etc/ssh/sshd.conf/%u# this is a comment",
			nil,
			"/etc/ssh/sshd.conf/%u",
			defaultSSHDPort,
			nil,
		},
		{
			"ignore port setting in sshd_config if preset",
			func(s *SSHManager) {
				s.sshdPort = 114
			},
			"Port 1030",
			nil,
			defaultAuthorizedKeysFile,
			114,
			nil,
		},
		{
			"should correctly parse port from Port config",
			nil,
			"Port 114",
			nil,
			defaultAuthorizedKeysFile,
			114,
			nil,
		},
		{
			"use the first one if multiple Port presented",
			nil,
			"Port 114 \t\n Port 1030 \t\n Port 215",
			nil,
			defaultAuthorizedKeysFile,
			114,
			nil,
		},
		{
			"should support parsing port from ListenAddress ipv4 address",
			nil,
			"ListenAddress 0.0.0.0:1030",
			nil,
			defaultAuthorizedKeysFile,
			1030,
			nil,
		},
		{
			"should support parsing port from ListenAddress ipv6 address",
			nil,
			"ListenAddress [2605:2700:0:3::4713:93e3]:215",
			nil,
			defaultAuthorizedKeysFile,
			215,
			nil,
		},
		{
			"should skip if ListenAddress ipv6 does not contain a port",
			nil,
			"ListenAddress 2605:2700:0:3::4713:93e3",
			nil,
			defaultAuthorizedKeysFile,
			defaultSSHDPort,
			nil,
		},
		{
			"should skip if ListenAddress ipv4 does not contain a port",
			nil,
			"ListenAddress 192.168.0.1",
			nil,
			defaultAuthorizedKeysFile,
			defaultSSHDPort,
			nil,
		},
		{
			"take the first occurrence if multiple ListenAddress presented",
			nil,
			"ListenAddress [2605:2700:0:3::4713:93e3]:215 \n\tListenAddress 0.0.0.0:1030 \n",
			nil,
			defaultAuthorizedKeysFile,
			215,
			nil,
		},
		{
			"take the first occurrence if both Port and ListenAddress presented",
			nil,
			"ListenAddress [2605:2700:0:3::4713:93e3]:215 \n\tPort 114 \n",
			nil,
			defaultAuthorizedKeysFile,
			215,
			nil,
		},
		{
			"should ignore invalid config",
			nil,
			"Port invalid \n Port# another invalid \n \nListenAddress [::]:not_a_valid_port\n ListenAddress [::]:114",
			nil,
			defaultAuthorizedKeysFile,
			114,
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
			if tt.prepare != nil {
				tt.prepare(s)
			}

			if err := s.parseSSHDConfig(); (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("parseSSHDConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if s.authorizedKeysFilePattern != tt.wantAuthorizedKeysFile {
				t.Errorf("parseSSHDConfig() AuthorizedKeysFile got = [%v], want [%v]", s.authorizedKeysFilePattern, tt.wantAuthorizedKeysFile)
			}
			if s.sshdPort != tt.wantSSHDPort {
				t.Errorf("parseSSHDConfig() SSHD Port got = [%v], want [%v]", s.sshdPort, tt.wantSSHDPort)
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
