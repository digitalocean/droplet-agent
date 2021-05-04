package actioner

import (
	"encoding/json"
	"testing"

	"github.com/digitalocean/dotty-agent/internal/sysaccess"

	"github.com/digitalocean/dotty-agent/internal/log"
	"github.com/digitalocean/dotty-agent/internal/metadata"
	"github.com/digitalocean/dotty-agent/internal/metadata/actioner/internal/mocks"
	"github.com/golang/mock/gomock"
)

func Test_dottyKeysActioner_do(t *testing.T) {
	log.Mute()

	validSSHKey1 := &sysaccess.SSHKey{
		OSUser:     "root",
		PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHxxGMc7paI72eTQSNoz+e9jxVZjYDsMwfy6MwPgZlzncKjm+QTfgilNEDskWfU8Om4EiOMedhvrDhBfVSbqAoA=",
		ActorEmail: "actor@email.com",
		TTL:        50,
	}
	validKey1Bytes, _ := json.Marshal(validSSHKey1)
	validKey1 := string(validKey1Bytes)
	validSSHKey2 := &sysaccess.SSHKey{
		OSUser:     "user2",
		PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHkfoI1jkzV53geVZ9IMvVA6uyMlYwDkHJw04LMDWuFgAsA/hiLcoRPW2T4/1b6YPLyBwbgjZXwZ31MyLWhKbLI=",
		ActorEmail: "actor2@email.com",
		TTL:        1800,
	}
	validKey2Bytes, _ := json.Marshal(validSSHKey2)
	validKey2 := string(validKey2Bytes)

	tests := []struct {
		name     string
		prepare  func(sshMgr *mocks.MocksshManager)
		metadata *metadata.Metadata
	}{
		{
			"should skip invalid raw keys",
			func(sshMgr *mocks.MocksshManager) {
				sshMgr.EXPECT().UpdateKeys([]*sysaccess.SSHKey{
					validSSHKey1,
					validSSHKey2,
				})
			},
			&metadata.Metadata{
				DOTTYKeys: []string{
					"invalid-key",
					validKey1,
					validKey2,
				},
			},
		},
		{
			"should allow to clear keys if metadata does not contain any valid keys",
			func(sshMgr *mocks.MocksshManager) {
				sshMgr.EXPECT().UpdateKeys([]*sysaccess.SSHKey{})
			},
			&metadata.Metadata{
				DOTTYKeys: []string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			sshMgrMock := mocks.NewMocksshManager(mockCtl)

			tt.prepare(sshMgrMock)
			da := &dottyKeysActioner{
				sshMgr: sshMgrMock,
			}
			da.do(tt.metadata)
		})
	}
}
