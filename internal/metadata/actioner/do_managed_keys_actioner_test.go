// SPDX-License-Identifier: Apache-2.0

package actioner

import (
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	"testing"

	"github.com/digitalocean/droplet-agent/internal/sysaccess"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/metadata"
	"github.com/digitalocean/droplet-agent/internal/metadata/actioner/internal/mocks"
)

func Test_dottyKeysActioner_do(t *testing.T) {
	log.Mute()
	newBoolPtr := func(v bool) *bool { return &v }

	validDOTTYKey1 := &sysaccess.SSHKey{
		OSUser:     "root",
		PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHxxGMc7paI72eTQSNoz+e9jxVZjYDsMwfy6MwPgZlzncKjm+QTfgilNEDskWfU8Om4EiOMedhvrDhBfVSbqAoA=",
		ActorEmail: "actor@email.com",
		TTL:        50,
		Type:       sysaccess.SSHKeyTypeDOTTY,
	}
	validDOTTYKey1Bytes, _ := json.Marshal(validDOTTYKey1)
	validDOTTYKey1Str := string(validDOTTYKey1Bytes)
	validDOTTYKey2 := &sysaccess.SSHKey{
		OSUser:     "user2",
		PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHkfoI1jkzV53geVZ9IMvVA6uyMlYwDkHJw04LMDWuFgAsA/hiLcoRPW2T4/1b6YPLyBwbgjZXwZ31MyLWhKbLI=",
		ActorEmail: "actor2@email.com",
		TTL:        1800,
		Type:       sysaccess.SSHKeyTypeDOTTY,
	}
	validDOTTYKey2Bytes, _ := json.Marshal(validDOTTYKey2)
	validDOTTYKey2Str := string(validDOTTYKey2Bytes)

	validDropletKey1 := &sysaccess.SSHKey{
		OSUser:    "foobar",
		PublicKey: "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBBFBd1GVYD8sCA0af8OnZmMAfD/pcecH2xiLt5+FzsJUdi27bhoQDsHn9JLVM1cG1yMtHh9lnGJ3OT6C3PoAwCw= -os_user=foobar",
		Type:      sysaccess.SSHKeyTypeDroplet,
	}
	validDropletKey1Str := "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBBFBd1GVYD8sCA0af8OnZmMAfD/pcecH2xiLt5+FzsJUdi27bhoQDsHn9JLVM1cG1yMtHh9lnGJ3OT6C3PoAwCw= -os_user=foobar"

	tests := []struct {
		name     string
		prepare  func(sshMgr *mocks.MocksshManager, keyParser *mocks.MocksshKeyParser)
		metadata *metadata.Metadata
	}{
		{
			"should skip invalid public keys",
			func(sshMgr *mocks.MocksshManager, keyParser *mocks.MocksshKeyParser) {
				sshMgr.EXPECT().DisableManagedDropletKeys()
				gomock.InOrder(
					keyParser.EXPECT().FromPublicKey("invalid-public-key").Return(nil, errors.New("oops")),
					keyParser.EXPECT().FromPublicKey(validDropletKey1Str).Return(validDropletKey1, nil),
				)
				sshMgr.EXPECT().UpdateKeys([]*sysaccess.SSHKey{
					validDropletKey1,
				}).Return(nil)
			},
			&metadata.Metadata{
				PublicKeys: []string{
					"invalid-public-key",
					validDropletKey1Str,
				},
			},
		},
		{
			"should skip invalid dotty keys",
			func(sshMgr *mocks.MocksshManager, keyParser *mocks.MocksshKeyParser) {
				sshMgr.EXPECT().DisableManagedDropletKeys()
				gomock.InOrder(
					keyParser.EXPECT().FromDOTTYKey("invalid-dotty-key").Return(nil, errors.New("oops")),
					keyParser.EXPECT().FromDOTTYKey(validDOTTYKey1Str).Return(validDOTTYKey1, nil),
				)
				sshMgr.EXPECT().UpdateKeys([]*sysaccess.SSHKey{
					validDOTTYKey1,
				}).Return(nil)
			},
			&metadata.Metadata{
				DOTTYKeys: []string{
					"invalid-dotty-key",
					validDOTTYKey1Str,
				},
			},
		},
		{
			"should combine public keys and dotty keys",
			func(sshMgr *mocks.MocksshManager, keyParser *mocks.MocksshKeyParser) {
				sshMgr.EXPECT().DisableManagedDropletKeys()
				gomock.InOrder(
					keyParser.EXPECT().FromPublicKey(validDropletKey1Str).Return(validDropletKey1, nil),
					keyParser.EXPECT().FromDOTTYKey(validDOTTYKey1Str).Return(validDOTTYKey1, nil),
					keyParser.EXPECT().FromDOTTYKey(validDOTTYKey2Str).Return(validDOTTYKey2, nil),
				)
				sshMgr.EXPECT().UpdateKeys([]*sysaccess.SSHKey{
					validDropletKey1,
					validDOTTYKey1,
					validDOTTYKey2,
				}).Return(nil)
			},
			&metadata.Metadata{
				PublicKeys: []string{
					validDropletKey1Str,
				},
				DOTTYKeys: []string{
					validDOTTYKey1Str,
					validDOTTYKey2Str,
				},
			},
		},
		{
			"should allow to clear keys if metadata does not contain any valid keys",
			func(sshMgr *mocks.MocksshManager, keyParser *mocks.MocksshKeyParser) {
				sshMgr.EXPECT().DisableManagedDropletKeys()
				sshMgr.EXPECT().UpdateKeys([]*sysaccess.SSHKey{})
			},
			&metadata.Metadata{
				PublicKeys: []string{},
				DOTTYKeys:  []string{},
			},
		},
		{
			"should turn on managed keys if specified",
			func(sshMgr *mocks.MocksshManager, keyParser *mocks.MocksshKeyParser) {
				sshMgr.EXPECT().EnableManagedDropletKeys()
				sshMgr.EXPECT().UpdateKeys([]*sysaccess.SSHKey{}).Return(nil)
			},
			&metadata.Metadata{
				ManagedKeysEnabled: newBoolPtr(true),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			sshMgrMock := mocks.NewMocksshManager(mockCtl)
			keyParserMock := mocks.NewMocksshKeyParser(mockCtl)

			tt.prepare(sshMgrMock, keyParserMock)
			da := &doManagedKeysActioner{
				sshMgr:    sshMgrMock,
				keyParser: keyParserMock,
			}
			da.do(tt.metadata)
		})
	}
}
