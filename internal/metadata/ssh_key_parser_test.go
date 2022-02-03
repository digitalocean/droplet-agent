package metadata

import (
	"fmt"
	"github.com/digitalocean/droplet-agent/internal/sysaccess"
	"reflect"
	"testing"
)

func TestSSHKeyParser_FromPublicKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		want    *sysaccess.SSHKey
		wantErr bool
	}{
		{
			"should support key without os_user flag",
			"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
			&sysaccess.SSHKey{
				PublicKey: "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
				Type:      sysaccess.SSHKeyTypeDroplet,
			},
			false,
		},
		{
			"should support key with os_user flag",
			"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE= -os_user=foo",
			&sysaccess.SSHKey{
				OSUser:    "foo",
				PublicKey: "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE= -os_user=foo",
				Type:      sysaccess.SSHKeyTypeDroplet,
			},
			false,
		},
		{
			"should pick the rightmost matched os_user flag if multiple presented",
			"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE= -os_user=foo comment1 -os_user=bar",
			&sysaccess.SSHKey{
				OSUser:    "bar",
				PublicKey: "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE= -os_user=foo comment1 -os_user=bar",
				Type:      sysaccess.SSHKeyTypeDroplet,
			},
			false,
		},
		{
			"should pick the rightmost valid os_user flag",
			"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE= -os_user=foo comment1 -os_user= ",
			&sysaccess.SSHKey{
				OSUser:    "foo",
				PublicKey: "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE= -os_user=foo comment1 -os_user=",
				Type:      sysaccess.SSHKeyTypeDroplet,
			},
			false,
		},
		{
			"should trim the key and cut the unnecessary prefix and suffix characters",
			"    \t \r \necdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE= comment1 someone@digitalocean.com \t\r\n",
			&sysaccess.SSHKey{
				PublicKey: "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE= comment1 someone@digitalocean.com",
				Type:      sysaccess.SSHKeyTypeDroplet,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewSSHKeyParser()
			got, err := p.FromPublicKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromPublicKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromPublicKey() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSHKeyParser_FromDOTTYKey(t *testing.T) {
	validSSHKey1 := &sysaccess.SSHKey{
		OSUser:     "root",
		PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHxxGMc7paI72eTQSNoz+e9jxVZjYDsMwfy6MwPgZlzncKjm+QTfgilNEDskWfU8Om4EiOMedhvrDhBfVSbqAoA=",
		ActorEmail: "actor@email.com",
		TTL:        50,
		Type:       sysaccess.SSHKeyTypeDOTTY,
	}
	validSSHKey1Str := "{\"os_user\":\"root\",\"ssh_key\":\"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHxxGMc7paI72eTQSNoz+e9jxVZjYDsMwfy6MwPgZlzncKjm+QTfgilNEDskWfU8Om4EiOMedhvrDhBfVSbqAoA=\",\"actor_email\":\"actor@email.com\",\"ttl\":50}"
	tests := []struct {
		name    string
		key     string
		want    *sysaccess.SSHKey
		wantErr bool
	}{
		{
			"happy path",
			validSSHKey1Str,
			validSSHKey1,
			false,
		},
		{
			"return error if not valid json",
			"invalid-json-string",
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &SSHKeyParser{}
			fmt.Println(tt.key)
			got, err := p.FromDOTTYKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromDOTTYKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromDOTTYKey() got = %v, want %v", got, tt.want)
			}
		})
	}
}
