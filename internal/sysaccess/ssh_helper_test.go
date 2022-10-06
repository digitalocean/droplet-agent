// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/sysaccess/internal/mocks"
	"github.com/digitalocean/droplet-agent/internal/sysutil"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/mock/gomock"
)

func Test_sshHelperImpl_authorizedKeysFile(t *testing.T) {
	tests := []struct {
		name              string
		authorizedKeyFile string
		user              *sysutil.User
		want              string
	}{
		{
			"resolve %% to %",
			"path/%%to/%%authorized_keys",
			&sysutil.User{},
			"path/%to/%authorized_keys",
		},
		{
			"resolve %h to user home dir",
			"%h/.ssh/authorized_keys",
			&sysutil.User{HomeDir: "/home/hlee"},
			"/home/hlee/.ssh/authorized_keys",
		},
		{
			"should strip the trailing slash of the home dir",
			"%h/.ssh/authorized_keys",
			&sysutil.User{HomeDir: "/home/hlee" + string(os.PathSeparator)},
			"/home/hlee/.ssh/authorized_keys",
		},
		{
			"resolve %u to user name",
			"/etc/ssh.d/%u/authorized_keys",
			&sysutil.User{Name: "hlee"},
			"/etc/ssh.d/hlee/authorized_keys",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &sshHelperImpl{
				mgr: &SSHManager{authorizedKeysFilePattern: tt.authorizedKeyFile},
			}
			if got := s.authorizedKeysFile(tt.user); got != tt.want {
				t.Errorf("authorizedKeysFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sshHelperImpl_prepareAuthorizedKeys(t *testing.T) {
	log.Mute()
	timeNow := time.Now()

	exampleKey1 := &SSHKey{
		OSUser:     "root",
		PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHxxGMc7paI72eTQSNoz+e9jxVZjYDsMwfy6MwPgZlzncKjm+QTfgilNEDskWfU8Om4EiOMedhvrDhBfVSbqAoA=",
		ActorEmail: "actor@email.com",
		TTL:        50,
		Type:       SSHKeyTypeDOTTY,
		expireAt:   timeNow.Add(10 * time.Second),
	}
	exampleKey2 := &SSHKey{
		OSUser:     "user2",
		PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHkfoI1jkzV53geVZ9IMvVA6uyMlYwDkHJw04LMDWuFgAsA/hiLcoRPW2T4/1b6YPLyBwbgjZXwZ31MyLWhKbLI=",
		ActorEmail: "actor2@email.com",
		TTL:        1800,
		Type:       SSHKeyTypeDOTTY,
		expireAt:   timeNow.Add(1800 * time.Second),
	}
	exampleKey3 := &SSHKey{
		OSUser:     "user3",
		PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHzeZZbcsOfu8hWB/OVntUCLZ1EWMiOU6BysslJIxe1mSnQzEjQBaMY/eK3vjipVIaktLLJ3FNCCXlFCPWFYkrs=",
		ActorEmail: "actor3@email.com",
		TTL:        1800,
		Type:       SSHKeyTypeDOTTY,
		expireAt:   timeNow.Add(300 * time.Second),
	}
	exampleKey4 := &SSHKey{
		OSUser:     "user4",
		PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBGpEBNmbOenW9wV5YM+HCR4Hc00IXM1NxW0/4Qkx9bZvKoFbFA0Vv9yLaFP7asvqXSPe7UnNwe9rXKDS4wlTXmI= \n",
		ActorEmail: "actor4@email.com",
		TTL:        1800,
		Type:       SSHKeyTypeDOTTY,
		expireAt:   timeNow.Add(900 * time.Second),
	}
	dropletKey1 := &SSHKey{
		OSUser:      "root",
		PublicKey:   "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
		Type:        SSHKeyTypeDroplet,
		fingerprint: "SHA256:w8bUbLGaB7nZg0zJisdljWq7HNMr+VOYXXVQU5nT1AI",
	}
	dropletKey2 := &SSHKey{
		OSUser:      "root",
		PublicKey:   "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBFWs0vUPi/q2dscBE5yzycy98ZSzs7kas5gNGrM62HGMUqM1lpO3nHbXqeBz/erOaPSoEk7TpR5wWMKYi6Yu3+Y=",
		Type:        SSHKeyTypeDroplet,
		fingerprint: "SHA256:8PEHs4nUAyUcVM6Fc6SVdaRhi6F55PiVFuh7oPH0Mgk",
	}
	type args struct {
		localKeys   []string
		managedKeys []*SSHKey
	}
	tests := []struct {
		name               string
		withoutManagedKeys bool
		args               args
		want               []string
	}{
		{
			name: "should remove all DO managed keys if managedKeys is empty",
			args: args{
				localKeys: []string{
					"# customer key 1",
					"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
					"# customer key 2",
					"",
					"",
					"",
					dropletKeyComment,
					dropletKeyFmt(dropletKey1),
					dottyComment,
					dottyKeyFmt(exampleKey2),
					"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCnMKX2t5cq+TE+CmpkD7Mbdb3CQE81xGzutwQkr91nz/EDDxOsBfYGUAuHH/7eb+JXno2LiU9sWO3w9/muSsP5zDXoZY9xCUuatvJsMBIUWC7O3uGeE0UJWpdkNpXrbo+IuU/1TsoKnDEMd3o5Etyq5rrotZ0/ap/q4JxkFmJCFpGwGMI5H+MWk0UXbVVDV6jn1YsvFuEZl9ju63AyGGfJU05O1HbW8E5VB0tXbQ2u1tuV8on2uG/3bc2JmRZ9C78kA5FwJUrDU1r41vqHFSFF1oTPHU1SWsSacr8FZ95/u0Hdh+c+FryUlVm8I+rptG9yeTvCKs+AtJv+BdhkZcW47ppMt2g702/gP9MphLVg04XKr6xP4Kj4Z+gjj+HEX5ucs9mkJwigeeoDm8lnydhOHzxdRnImW3E7lksTyQRw+fgzJ8hFcxA5J7G4O7xuypAWp/vmzaOUrwMq741WRMJEwEo0cGL7P8nGw/BQA6h7BWb7VA4mvtOxVkBcolVUQ2FpatBaSkdr2EEvCq5dZddroGi2OaPvEgUe6cl22JA6tv2Ah/k6q5NgR2Qik+jCOKSSUkQrVA6/eGJz3Rt9zf99Ah3hzHPEVpX6IVpKOMZUa66pw+bFLJLonzV2cGu/nQn0KCtI7AcoB+GWyqm1oqRDwzmCwqJRXJJ0PovKrSVHPQ== customer@key2",
					"# customer key 3",
					"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBDdPvHGQm4OWJd9vDvz405D7BFxhwu09IvnPOf0+e/nrGzWykXJsm9Hy1AdjSM7lgUEleeOQeMZt7EIlZJ8Eou4= customer@key3",
				},
				managedKeys: []*SSHKey{},
			},
			want: []string{
				"# customer key 1",
				"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
				"# customer key 2",
				"",
				"",
				"",
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCnMKX2t5cq+TE+CmpkD7Mbdb3CQE81xGzutwQkr91nz/EDDxOsBfYGUAuHH/7eb+JXno2LiU9sWO3w9/muSsP5zDXoZY9xCUuatvJsMBIUWC7O3uGeE0UJWpdkNpXrbo+IuU/1TsoKnDEMd3o5Etyq5rrotZ0/ap/q4JxkFmJCFpGwGMI5H+MWk0UXbVVDV6jn1YsvFuEZl9ju63AyGGfJU05O1HbW8E5VB0tXbQ2u1tuV8on2uG/3bc2JmRZ9C78kA5FwJUrDU1r41vqHFSFF1oTPHU1SWsSacr8FZ95/u0Hdh+c+FryUlVm8I+rptG9yeTvCKs+AtJv+BdhkZcW47ppMt2g702/gP9MphLVg04XKr6xP4Kj4Z+gjj+HEX5ucs9mkJwigeeoDm8lnydhOHzxdRnImW3E7lksTyQRw+fgzJ8hFcxA5J7G4O7xuypAWp/vmzaOUrwMq741WRMJEwEo0cGL7P8nGw/BQA6h7BWb7VA4mvtOxVkBcolVUQ2FpatBaSkdr2EEvCq5dZddroGi2OaPvEgUe6cl22JA6tv2Ah/k6q5NgR2Qik+jCOKSSUkQrVA6/eGJz3Rt9zf99Ah3hzHPEVpX6IVpKOMZUa66pw+bFLJLonzV2cGu/nQn0KCtI7AcoB+GWyqm1oqRDwzmCwqJRXJJ0PovKrSVHPQ== customer@key2",
				"# customer key 3",
				"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBDdPvHGQm4OWJd9vDvz405D7BFxhwu09IvnPOf0+e/nrGzWykXJsm9Hy1AdjSM7lgUEleeOQeMZt7EIlZJ8Eou4= customer@key3",
			},
		},
		{
			name: "should keep all existing droplet keys and remove dotty keys if managedKeys is nil",
			args: args{
				localKeys: []string{
					"# customer key 1",
					"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
					dropletKeyComment,
					dropletKeyFmt(dropletKey1),
					dottyComment,
					dottyKeyFmt(exampleKey2),
					dropletKey2.PublicKey,
				},
				managedKeys: nil,
			},
			want: []string{
				"# customer key 1",
				"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
				dropletKeyComment,
				dropletKeyFmt(dropletKey1),
				dropletKey2.PublicKey,
			},
		},
		{
			name:               "should still keep all existing droplet keys and remove dotty keys if configured to not manage ssh keys and managedKeys is nil",
			withoutManagedKeys: true,
			args: args{
				localKeys: []string{
					"# customer key 1",
					"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
					dropletKeyComment,
					dropletKeyFmt(dropletKey1),
					dottyComment,
					dottyKeyFmt(exampleKey2),
					dropletKey2.PublicKey,
				},
				managedKeys: nil,
			},
			want: []string{
				"# customer key 1",
				"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
				dropletKeyComment,
				dropletKeyFmt(dropletKey1),
				dropletKey2.PublicKey,
			},
		},
		{
			name: "should recognize DO managed keys comments case-agnostic",
			args: args{
				localKeys: []string{
					"# ADDED and Managed by DigitalOcean TTY service (DoTTY)",
					dottyKeyFmt(exampleKey1),
					"# ManAged through DIGITALOcEaN",
					dropletKeyFmt(dropletKey1),
					strings.ToTitle(dottyComment),
					dottyKeyFmt(exampleKey2),
				},
				managedKeys: []*SSHKey{
					dropletKey1,
					exampleKey1,
					exampleKey2,
				},
			},
			want: []string{
				dropletKeyComment,
				dropletKeyFmt(dropletKey1),
				dottyComment,
				dottyKeyFmt(exampleKey1),
				dottyComment,
				dottyKeyFmt(exampleKey2),
			},
		},
		{
			name: "should append all dotty keys after the customer's keys and properly reformat",
			args: args{
				localKeys: []string{
					"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
					dropletKeyFmt(dropletKey1),
					"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCnMKX2t5cq+TE+CmpkD7Mbdb3CQE81xGzutwQkr91nz/EDDxOsBfYGUAuHH/7eb+JXno2LiU9sWO3w9/muSsP5zDXoZY9xCUuatvJsMBIUWC7O3uGeE0UJWpdkNpXrbo+IuU/1TsoKnDEMd3o5Etyq5rrotZ0/ap/q4JxkFmJCFpGwGMI5H+MWk0UXbVVDV6jn1YsvFuEZl9ju63AyGGfJU05O1HbW8E5VB0tXbQ2u1tuV8on2uG/3bc2JmRZ9C78kA5FwJUrDU1r41vqHFSFF1oTPHU1SWsSacr8FZ95/u0Hdh+c+FryUlVm8I+rptG9yeTvCKs+AtJv+BdhkZcW47ppMt2g702/gP9MphLVg04XKr6xP4Kj4Z+gjj+HEX5ucs9mkJwigeeoDm8lnydhOHzxdRnImW3E7lksTyQRw+fgzJ8hFcxA5J7G4O7xuypAWp/vmzaOUrwMq741WRMJEwEo0cGL7P8nGw/BQA6h7BWb7VA4mvtOxVkBcolVUQ2FpatBaSkdr2EEvCq5dZddroGi2OaPvEgUe6cl22JA6tv2Ah/k6q5NgR2Qik+jCOKSSUkQrVA6/eGJz3Rt9zf99Ah3hzHPEVpX6IVpKOMZUa66pw+bFLJLonzV2cGu/nQn0KCtI7AcoB+GWyqm1oqRDwzmCwqJRXJJ0PovKrSVHPQ== customer@key2",
					"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBDdPvHGQm4OWJd9vDvz405D7BFxhwu09IvnPOf0+e/nrGzWykXJsm9Hy1AdjSM7lgUEleeOQeMZt7EIlZJ8Eou4= customer@key3",
				},
				managedKeys: []*SSHKey{
					exampleKey1,
					dropletKey1,
				},
			},
			want: []string{
				"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCnMKX2t5cq+TE+CmpkD7Mbdb3CQE81xGzutwQkr91nz/EDDxOsBfYGUAuHH/7eb+JXno2LiU9sWO3w9/muSsP5zDXoZY9xCUuatvJsMBIUWC7O3uGeE0UJWpdkNpXrbo+IuU/1TsoKnDEMd3o5Etyq5rrotZ0/ap/q4JxkFmJCFpGwGMI5H+MWk0UXbVVDV6jn1YsvFuEZl9ju63AyGGfJU05O1HbW8E5VB0tXbQ2u1tuV8on2uG/3bc2JmRZ9C78kA5FwJUrDU1r41vqHFSFF1oTPHU1SWsSacr8FZ95/u0Hdh+c+FryUlVm8I+rptG9yeTvCKs+AtJv+BdhkZcW47ppMt2g702/gP9MphLVg04XKr6xP4Kj4Z+gjj+HEX5ucs9mkJwigeeoDm8lnydhOHzxdRnImW3E7lksTyQRw+fgzJ8hFcxA5J7G4O7xuypAWp/vmzaOUrwMq741WRMJEwEo0cGL7P8nGw/BQA6h7BWb7VA4mvtOxVkBcolVUQ2FpatBaSkdr2EEvCq5dZddroGi2OaPvEgUe6cl22JA6tv2Ah/k6q5NgR2Qik+jCOKSSUkQrVA6/eGJz3Rt9zf99Ah3hzHPEVpX6IVpKOMZUa66pw+bFLJLonzV2cGu/nQn0KCtI7AcoB+GWyqm1oqRDwzmCwqJRXJJ0PovKrSVHPQ== customer@key2",
				"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBDdPvHGQm4OWJd9vDvz405D7BFxhwu09IvnPOf0+e/nrGzWykXJsm9Hy1AdjSM7lgUEleeOQeMZt7EIlZJ8Eou4= customer@key3",
				dottyComment,
				dottyKeyFmt(exampleKey1),
				dropletKeyComment,
				dropletKeyFmt(dropletKey1),
			},
		},
		{
			name: "should recognize droplet keys from existing keys",
			args: args{
				localKeys: []string{
					dropletKey1.PublicKey + " comment foobar",
				},
				managedKeys: []*SSHKey{
					dropletKey1,
				},
			},
			want: []string{
				dropletKeyComment,
				dropletKeyFmt(dropletKey1),
			},
		},
		{
			name:               "should not attempt to recognize droplet keys if configured not to manage droplet ssh keys",
			withoutManagedKeys: true,
			args: args{
				localKeys: []string{
					dropletKey1.PublicKey + " comment foobar",
				},
				managedKeys: []*SSHKey{},
			},
			want: []string{
				dropletKey1.PublicKey + " comment foobar",
			},
		},
		{
			name: "should remove dotty keys that are not in the given list",
			args: args{
				localKeys: []string{
					"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
					dottyComment,
					dottyKeyFmt(exampleKey2),
					"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCnMKX2t5cq+TE+CmpkD7Mbdb3CQE81xGzutwQkr91nz/EDDxOsBfYGUAuHH/7eb+JXno2LiU9sWO3w9/muSsP5zDXoZY9xCUuatvJsMBIUWC7O3uGeE0UJWpdkNpXrbo+IuU/1TsoKnDEMd3o5Etyq5rrotZ0/ap/q4JxkFmJCFpGwGMI5H+MWk0UXbVVDV6jn1YsvFuEZl9ju63AyGGfJU05O1HbW8E5VB0tXbQ2u1tuV8on2uG/3bc2JmRZ9C78kA5FwJUrDU1r41vqHFSFF1oTPHU1SWsSacr8FZ95/u0Hdh+c+FryUlVm8I+rptG9yeTvCKs+AtJv+BdhkZcW47ppMt2g702/gP9MphLVg04XKr6xP4Kj4Z+gjj+HEX5ucs9mkJwigeeoDm8lnydhOHzxdRnImW3E7lksTyQRw+fgzJ8hFcxA5J7G4O7xuypAWp/vmzaOUrwMq741WRMJEwEo0cGL7P8nGw/BQA6h7BWb7VA4mvtOxVkBcolVUQ2FpatBaSkdr2EEvCq5dZddroGi2OaPvEgUe6cl22JA6tv2Ah/k6q5NgR2Qik+jCOKSSUkQrVA6/eGJz3Rt9zf99Ah3hzHPEVpX6IVpKOMZUa66pw+bFLJLonzV2cGu/nQn0KCtI7AcoB+GWyqm1oqRDwzmCwqJRXJJ0PovKrSVHPQ== customer@key2",
					dottyComment,
					dottyKeyFmt(exampleKey3),
					"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBDdPvHGQm4OWJd9vDvz405D7BFxhwu09IvnPOf0+e/nrGzWykXJsm9Hy1AdjSM7lgUEleeOQeMZt7EIlZJ8Eou4= customer@key3",
				},
				managedKeys: []*SSHKey{
					exampleKey1,
					exampleKey2,
					exampleKey4,
				},
			},
			want: []string{
				"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCnMKX2t5cq+TE+CmpkD7Mbdb3CQE81xGzutwQkr91nz/EDDxOsBfYGUAuHH/7eb+JXno2LiU9sWO3w9/muSsP5zDXoZY9xCUuatvJsMBIUWC7O3uGeE0UJWpdkNpXrbo+IuU/1TsoKnDEMd3o5Etyq5rrotZ0/ap/q4JxkFmJCFpGwGMI5H+MWk0UXbVVDV6jn1YsvFuEZl9ju63AyGGfJU05O1HbW8E5VB0tXbQ2u1tuV8on2uG/3bc2JmRZ9C78kA5FwJUrDU1r41vqHFSFF1oTPHU1SWsSacr8FZ95/u0Hdh+c+FryUlVm8I+rptG9yeTvCKs+AtJv+BdhkZcW47ppMt2g702/gP9MphLVg04XKr6xP4Kj4Z+gjj+HEX5ucs9mkJwigeeoDm8lnydhOHzxdRnImW3E7lksTyQRw+fgzJ8hFcxA5J7G4O7xuypAWp/vmzaOUrwMq741WRMJEwEo0cGL7P8nGw/BQA6h7BWb7VA4mvtOxVkBcolVUQ2FpatBaSkdr2EEvCq5dZddroGi2OaPvEgUe6cl22JA6tv2Ah/k6q5NgR2Qik+jCOKSSUkQrVA6/eGJz3Rt9zf99Ah3hzHPEVpX6IVpKOMZUa66pw+bFLJLonzV2cGu/nQn0KCtI7AcoB+GWyqm1oqRDwzmCwqJRXJJ0PovKrSVHPQ== customer@key2",
				"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBDdPvHGQm4OWJd9vDvz405D7BFxhwu09IvnPOf0+e/nrGzWykXJsm9Hy1AdjSM7lgUEleeOQeMZt7EIlZJ8Eou4= customer@key3",
				dottyComment,
				dottyKeyFmt(exampleKey1),
				dottyComment,
				dottyKeyFmt(exampleKey2),
				dottyComment,
				dottyKeyFmt(exampleKey4),
			},
		},
		{
			name: "should remove droplet keys that are not in the given list",
			args: args{
				localKeys: []string{
					"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
					dropletKeyComment,
					dropletKeyFmt(dropletKey1),
					dropletKey2.PublicKey,
				},
				managedKeys: []*SSHKey{
					dropletKey2,
				},
			},
			want: []string{
				"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
				dropletKeyComment,
				dropletKeyFmt(dropletKey2),
			},
		},
		{
			name: "should properly handle comments and empty lines",
			args: args{
				localKeys: []string{
					"#comment 1",
					"",
					"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
					dottyComment,
					"# added comment (will not be kept in the same place)",
					"",
					dottyKeyFmt(exampleKey2),
					"# another comment",
					"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCnMKX2t5cq+TE+CmpkD7Mbdb3CQE81xGzutwQkr91nz/EDDxOsBfYGUAuHH/7eb+JXno2LiU9sWO3w9/muSsP5zDXoZY9xCUuatvJsMBIUWC7O3uGeE0UJWpdkNpXrbo+IuU/1TsoKnDEMd3o5Etyq5rrotZ0/ap/q4JxkFmJCFpGwGMI5H+MWk0UXbVVDV6jn1YsvFuEZl9ju63AyGGfJU05O1HbW8E5VB0tXbQ2u1tuV8on2uG/3bc2JmRZ9C78kA5FwJUrDU1r41vqHFSFF1oTPHU1SWsSacr8FZ95/u0Hdh+c+FryUlVm8I+rptG9yeTvCKs+AtJv+BdhkZcW47ppMt2g702/gP9MphLVg04XKr6xP4Kj4Z+gjj+HEX5ucs9mkJwigeeoDm8lnydhOHzxdRnImW3E7lksTyQRw+fgzJ8hFcxA5J7G4O7xuypAWp/vmzaOUrwMq741WRMJEwEo0cGL7P8nGw/BQA6h7BWb7VA4mvtOxVkBcolVUQ2FpatBaSkdr2EEvCq5dZddroGi2OaPvEgUe6cl22JA6tv2Ah/k6q5NgR2Qik+jCOKSSUkQrVA6/eGJz3Rt9zf99Ah3hzHPEVpX6IVpKOMZUa66pw+bFLJLonzV2cGu/nQn0KCtI7AcoB+GWyqm1oqRDwzmCwqJRXJJ0PovKrSVHPQ== customer@key2",
					"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBDdPvHGQm4OWJd9vDvz405D7BFxhwu09IvnPOf0+e/nrGzWykXJsm9Hy1AdjSM7lgUEleeOQeMZt7EIlZJ8Eou4= customer@key3",
				},
				managedKeys: []*SSHKey{
					exampleKey1,
					exampleKey2,
				},
			},
			want: []string{
				"#comment 1",
				"",
				"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHeAQeGsd93e5G41zQ3/N1rQ9OT5cj5xLwD0q7sf6fLFdMiDdxVIRFt/Qv+dCvvvZ3xO+Ers7aemTnEivfJSadU= customer@key1",
				"# added comment (will not be kept in the same place)",
				"",
				"# another comment",
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCnMKX2t5cq+TE+CmpkD7Mbdb3CQE81xGzutwQkr91nz/EDDxOsBfYGUAuHH/7eb+JXno2LiU9sWO3w9/muSsP5zDXoZY9xCUuatvJsMBIUWC7O3uGeE0UJWpdkNpXrbo+IuU/1TsoKnDEMd3o5Etyq5rrotZ0/ap/q4JxkFmJCFpGwGMI5H+MWk0UXbVVDV6jn1YsvFuEZl9ju63AyGGfJU05O1HbW8E5VB0tXbQ2u1tuV8on2uG/3bc2JmRZ9C78kA5FwJUrDU1r41vqHFSFF1oTPHU1SWsSacr8FZ95/u0Hdh+c+FryUlVm8I+rptG9yeTvCKs+AtJv+BdhkZcW47ppMt2g702/gP9MphLVg04XKr6xP4Kj4Z+gjj+HEX5ucs9mkJwigeeoDm8lnydhOHzxdRnImW3E7lksTyQRw+fgzJ8hFcxA5J7G4O7xuypAWp/vmzaOUrwMq741WRMJEwEo0cGL7P8nGw/BQA6h7BWb7VA4mvtOxVkBcolVUQ2FpatBaSkdr2EEvCq5dZddroGi2OaPvEgUe6cl22JA6tv2Ah/k6q5NgR2Qik+jCOKSSUkQrVA6/eGJz3Rt9zf99Ah3hzHPEVpX6IVpKOMZUa66pw+bFLJLonzV2cGu/nQn0KCtI7AcoB+GWyqm1oqRDwzmCwqJRXJJ0PovKrSVHPQ== customer@key2",
				"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBDdPvHGQm4OWJd9vDvz405D7BFxhwu09IvnPOf0+e/nrGzWykXJsm9Hy1AdjSM7lgUEleeOQeMZt7EIlZJ8Eou4= customer@key3",
				dottyComment,
				dottyKeyFmt(exampleKey1),
				dottyComment,
				dottyKeyFmt(exampleKey2),
			},
		},
		{
			name: "should okay if local keys empty",
			args: args{
				localKeys: nil,
				managedKeys: []*SSHKey{
					exampleKey1,
					exampleKey2,
					exampleKey3,
					exampleKey4,
				},
			},
			want: []string{
				dottyComment,
				dottyKeyFmt(exampleKey1),
				dottyComment,
				dottyKeyFmt(exampleKey2),
				dottyComment,
				dottyKeyFmt(exampleKey3),
				dottyComment,
				dottyKeyFmt(exampleKey4),
			},
		},
		{
			name: "should return empty slice if both local keys and dotty are empty",
			args: args{
				localKeys:   nil,
				managedKeys: nil,
			},
			want: []string{},
		},
		{
			name:               "should not attempt to remove droplet keys if configured not to manage droplet ssh keys",
			withoutManagedKeys: true,
			args: args{
				localKeys: []string{
					dropletKeyComment,
					dropletKeyFmt(dropletKey1),
				},
				managedKeys: []*SSHKey{},
			},
			want: []string{
				dropletKeyComment,
				dropletKeyFmt(dropletKey1),
			},
		},
		{
			name:               "should still remove dotty keys if configured not to manage droplet ssh keys",
			withoutManagedKeys: true,
			args: args{
				localKeys: []string{
					dottyComment,
					dottyKeyFmt(exampleKey1),
				},
				managedKeys: []*SSHKey{},
			},
			want: []string{},
		},
		{
			name:               "should not attempt to add droplet keys if configured not to manage droplet ssh keys",
			withoutManagedKeys: true,
			args: args{
				localKeys: []string{},
				managedKeys: []*SSHKey{
					dropletKey1,
				},
			},
			want: []string{},
		},
		{
			name:               "should still support adding dotty keys if configured not to manage droplet ssh keys",
			withoutManagedKeys: true,
			args: args{
				localKeys: []string{},
				managedKeys: []*SSHKey{
					dropletKey1,
					exampleKey1,
				},
			},
			want: []string{
				dottyComment,
				dottyKeyFmt(exampleKey1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &sshHelperImpl{
				timeNow: func() time.Time {
					return timeNow
				},
				mgr: &SSHManager{
					manageDropletKeys: manageDropletKeysEnabled,
				},
			}
			if tt.withoutManagedKeys {
				s.mgr.manageDropletKeys = manageDropletKeysDisabled
			}
			if got := s.prepareAuthorizedKeys(tt.args.localKeys, tt.args.managedKeys); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prepareAuthorizedKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dottyKeyFmt(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		key      *SSHKey
		wantInfo *sshKeyInfo
	}{
		{
			"full info",
			&SSHKey{
				OSUser:     "root",
				PublicKey:  "alg base64-key",
				ActorEmail: "actor@email.com",
				TTL:        50,
				expireAt:   now.Add(20 * time.Second),
			},
			&sshKeyInfo{
				OSUser:     "root",
				ActorEmail: "actor@email.com",
				ExpireAt:   now.Add(20 * time.Second).Format(time.RFC3339),
			},
		},
		{
			"no os user",
			&SSHKey{
				PublicKey:  "alg base64-key",
				ActorEmail: "actor@email.com",
				TTL:        50,
				expireAt:   now.Add(15 * time.Second),
			},
			&sshKeyInfo{
				ActorEmail: "actor@email.com",
				ExpireAt:   now.Add(15 * time.Second).Format(time.RFC3339),
			},
		},
		{
			"no actor email",
			&SSHKey{
				OSUser:    "root",
				PublicKey: "alg base64-key",
				TTL:       50,
				expireAt:  now.Add(10 * time.Second),
			},
			&sshKeyInfo{
				OSUser:   "root",
				ExpireAt: now.Add(10 * time.Second).Format(time.RFC3339),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dottyKeyFmt(tt.key)
			lineEnd := "-" + dottyKeyIndicator
			if !strings.HasSuffix(got, lineEnd) {
				t.Errorf("dottyKeyFmt() missing dotty key indicator")
			}
			if !strings.HasPrefix(got, tt.key.PublicKey) {
				t.Errorf("dottyKeyFmt() missing key: %v", tt.key.PublicKey)
			}

			info := &sshKeyInfo{}
			c := got[len(tt.key.PublicKey) : len(got)-len(lineEnd)]
			if err := json.Unmarshal([]byte(c), info); err != nil {
				t.Errorf("dottyKeyFmt() unexpected key comment. %s, %v", c, err)
			}
			expectedInfo := &sshKeyInfo{
				OSUser:     tt.key.OSUser,
				ActorEmail: tt.key.ActorEmail,
				ExpireAt:   tt.key.expireAt.Format(time.RFC3339),
			}
			if !reflect.DeepEqual(expectedInfo, info) {
				t.Errorf("dottyKeyFmt() = %v, want %v", info, expectedInfo)
			}

		})
	}
}

func Test_areSameKeys(t *testing.T) {
	key1 := &SSHKey{
		OSUser:     "root",
		PublicKey:  "public-key-1",
		ActorEmail: "actor-email-1",
		TTL:        25,
	}
	key11 := &SSHKey{
		OSUser:     "root",
		PublicKey:  "public-key-1",
		ActorEmail: "actor-email-11",
		TTL:        255,
	}
	key2 := &SSHKey{
		OSUser:     "root",
		PublicKey:  "public-key-2",
		ActorEmail: "actor-email-2",
		TTL:        25,
	}
	key3 := &SSHKey{
		OSUser:     "root",
		PublicKey:  "public-key-3",
		ActorEmail: "actor-email-3",
		TTL:        25,
	}
	type args struct {
		keys1 []*SSHKey
		keys2 []*SSHKey
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"should return true if both are nil slice",
			args{
				keys1: nil,
				keys2: nil,
			},
			true,
		},
		{
			"should return false if comparing between a nil slice and an empty slice",
			args{
				keys1: nil,
				keys2: []*SSHKey{},
			},
			false,
		},
		{
			"should return false if length not equal",
			args{
				keys1: []*SSHKey{key1, key2, key3},
				keys2: []*SSHKey{key1, key2},
			},
			false,
		},
		{
			"should return true regardless the order",
			args{
				keys1: []*SSHKey{key1, key2, key3},
				keys2: []*SSHKey{key2, key3, key1},
			},
			true,
		},
		{
			"should support duplicated entries",
			args{
				keys1: []*SSHKey{key1, key2, key3, key1},
				keys2: []*SSHKey{key2, key3, key1, key1},
			},
			true,
		},
		{
			"should properly handle duplicated entries",
			args{
				keys1: []*SSHKey{key1, key1, key2},
				keys2: []*SSHKey{key2, key2, key1},
			},
			false,
		},
		{
			"should check all value in s2 exists in s1",
			args{
				keys1: []*SSHKey{key1, key1, key1},
				keys2: []*SSHKey{key2, key3, key1},
			},
			false,
		},
		{
			"should check all value in s1 exists in s2",
			args{
				keys1: []*SSHKey{key2, key3, key1},
				keys2: []*SSHKey{key1, key1, key1},
			},
			false,
		},
		{
			"should consider equal as long as keys have same os_user and public_key",
			args{
				keys1: []*SSHKey{key1},
				keys2: []*SSHKey{key11},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &sshHelperImpl{}
			if got := s.areSameKeys(tt.args.keys1, tt.args.keys2); got != tt.want {
				t.Errorf("areSameKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sshHelperImpl_removeExpiredKeys(t *testing.T) {
	timeNow := time.Now()

	tests := []struct {
		name             string
		originalKeys     map[string][]*SSHKey
		wantFilteredKeys map[string][]*SSHKey
	}{
		{
			"should support the case when originalKeys is nil",
			nil,
			nil,
		},
		{
			"should support the case when originalKeys is empty",
			map[string][]*SSHKey{},
			map[string][]*SSHKey{},
		},
		{
			"should remove expired keys",
			map[string][]*SSHKey{
				"user1": {
					&SSHKey{
						OSUser:    "user1",
						PublicKey: "valid-key-1",
						expireAt:  timeNow.Add(50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
					&SSHKey{
						OSUser:    "user1",
						PublicKey: "expired-key-2",
						expireAt:  timeNow.Add(-50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
					&SSHKey{
						OSUser:    "user1",
						PublicKey: "valid-key-3",
						expireAt:  timeNow.Add(50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
				},
				"user2": {
					&SSHKey{
						OSUser:    "user2",
						PublicKey: "expired-key-1",
						expireAt:  timeNow.Add(-50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
					&SSHKey{
						OSUser:    "user2",
						PublicKey: "valid-key-2",
						expireAt:  timeNow.Add(50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
				},
			},
			map[string][]*SSHKey{
				"user1": {
					&SSHKey{
						OSUser:    "user1",
						PublicKey: "valid-key-1",
						expireAt:  timeNow.Add(50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
					&SSHKey{
						OSUser:    "user1",
						PublicKey: "valid-key-3",
						expireAt:  timeNow.Add(50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
				},
				"user2": {
					&SSHKey{
						OSUser:    "user2",
						PublicKey: "valid-key-2",
						expireAt:  timeNow.Add(50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
				},
			},
		},
		{
			"should not remove DOManaged keys",
			map[string][]*SSHKey{
				"user1": {
					&SSHKey{
						OSUser:    "user1",
						PublicKey: "expired-key-1",
						Type:      SSHKeyTypeDroplet,
						expireAt:  timeNow.Add(-50 * time.Second),
					},
				},
				"user2": {
					&SSHKey{
						OSUser:    "user2",
						PublicKey: "expired-key-2",
						Type:      SSHKeyTypeDroplet,
						expireAt:  timeNow.Add(-50 * time.Second),
					},
				},
			},
			map[string][]*SSHKey{
				"user1": {
					&SSHKey{
						OSUser:    "user1",
						PublicKey: "expired-key-1",
						Type:      SSHKeyTypeDroplet,
						expireAt:  timeNow.Add(-50 * time.Second),
					},
				},
				"user2": {
					&SSHKey{
						OSUser:    "user2",
						PublicKey: "expired-key-2",
						Type:      SSHKeyTypeDroplet,
						expireAt:  timeNow.Add(-50 * time.Second),
					},
				},
			},
		},
		{
			"should remove user if all keys expired",
			map[string][]*SSHKey{
				"user1": {
					&SSHKey{
						OSUser:    "user1",
						PublicKey: "expired-key-1",
						expireAt:  timeNow.Add(-50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
					&SSHKey{
						OSUser:    "user1",
						PublicKey: "expired-key-2",
						expireAt:  timeNow.Add(-50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
					&SSHKey{
						OSUser:    "user1",
						PublicKey: "expired-key-3",
						expireAt:  timeNow.Add(-50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
				},
				"user2": {
					&SSHKey{
						OSUser:    "user2",
						PublicKey: "expired-key-1",
						expireAt:  timeNow.Add(-50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
					&SSHKey{
						OSUser:    "user2",
						PublicKey: "valid-key-2",
						expireAt:  timeNow.Add(50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
				},
			},
			map[string][]*SSHKey{
				"user2": {
					&SSHKey{
						OSUser:    "user2",
						PublicKey: "valid-key-2",
						expireAt:  timeNow.Add(50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
				},
			},
		},
		{
			"should remove user with empty list",
			map[string][]*SSHKey{
				"user1": {},
				"user2": {
					&SSHKey{
						OSUser:    "user2",
						PublicKey: "expired-key-1",
						expireAt:  timeNow.Add(-50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
					&SSHKey{
						OSUser:    "user2",
						PublicKey: "valid-key-2",
						expireAt:  timeNow.Add(50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
				},
			},
			map[string][]*SSHKey{
				"user2": {
					&SSHKey{
						OSUser:    "user2",
						PublicKey: "valid-key-2",
						expireAt:  timeNow.Add(50 * time.Second),
						Type:      SSHKeyTypeDOTTY,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &sshHelperImpl{
				timeNow: func() time.Time {
					return timeNow
				},
			}
			if gotFilteredKeys := s.removeExpiredKeys(tt.originalKeys); !reflect.DeepEqual(gotFilteredKeys, tt.wantFilteredKeys) {
				t.Errorf("removeExpiredKeys() = %v, want %v", gotFilteredKeys, tt.wantFilteredKeys)
			}
		})
	}
}

func Test_sshHelperImpl_validateKey(t *testing.T) {
	timeNow := time.Now()
	tests := []struct {
		name    string
		key     *SSHKey
		wantKey *SSHKey
		wantErr error
	}{
		{
			"should set OSUser to default if empty",
			&SSHKey{
				OSUser:     "",
				PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
				ActorEmail: "actor@email.com",
				Type:       SSHKeyTypeDOTTY,
				TTL:        60,
			},
			&SSHKey{
				OSUser:      defaultOSUser,
				PublicKey:   "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
				ActorEmail:  "actor@email.com",
				Type:        SSHKeyTypeDOTTY,
				TTL:         60,
				fingerprint: "SHA256:w8bUbLGaB7nZg0zJisdljWq7HNMr+VOYXXVQU5nT1AI",
				expireAt:    timeNow.Add(60 * time.Second),
			},
			nil,
		},
		{
			"invalid ttl",
			&SSHKey{
				OSUser:     "root",
				PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
				ActorEmail: "actor@email.com",
				Type:       SSHKeyTypeDOTTY,
				TTL:        0,
			},
			&SSHKey{
				OSUser:     "root",
				PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
				ActorEmail: "actor@email.com",
				Type:       SSHKeyTypeDOTTY,
				TTL:        0,
			},
			ErrInvalidKey,
		},
		{
			"should not check ttl for DOManaged keys",
			&SSHKey{
				OSUser:     "root",
				PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
				ActorEmail: "actor@email.com",
				Type:       SSHKeyTypeDroplet,
				TTL:        0,
			},
			&SSHKey{
				OSUser:      "root",
				PublicKey:   "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
				ActorEmail:  "actor@email.com",
				Type:        SSHKeyTypeDroplet,
				fingerprint: "SHA256:w8bUbLGaB7nZg0zJisdljWq7HNMr+VOYXXVQU5nT1AI",
				TTL:         0,
			},
			nil,
		},
		{
			"invalid public key",
			&SSHKey{
				OSUser:     "root",
				PublicKey:  "not a valid ssh key",
				ActorEmail: "actor@email.com",
				Type:       SSHKeyTypeDroplet,
				TTL:        50,
			},
			&SSHKey{
				OSUser:     "root",
				PublicKey:  "not a valid ssh key",
				ActorEmail: "actor@email.com",
				Type:       SSHKeyTypeDroplet,
				TTL:        50,
			},
			ErrInvalidKey,
		},
		{
			"should properly set the expire time of the key if it's a dotty key",
			&SSHKey{
				OSUser:     "user1",
				PublicKey:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
				ActorEmail: "actor@email.com",
				Type:       SSHKeyTypeDOTTY,
				TTL:        60,
			},
			&SSHKey{
				OSUser:      "user1",
				PublicKey:   "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHRjqHzBANlihrvlhyecJecbR4yV5ufOgl9fllxDFpDGMMDd6Pb+ypR/noxmQwa9ik8Z3ki9e1UAIeQ8K5R3kpE=",
				ActorEmail:  "actor@email.com",
				TTL:         60,
				Type:        SSHKeyTypeDOTTY,
				fingerprint: "SHA256:w8bUbLGaB7nZg0zJisdljWq7HNMr+VOYXXVQU5nT1AI",
				expireAt:    timeNow.Add(60 * time.Second),
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &sshHelperImpl{
				timeNow: func() time.Time {
					return timeNow
				},
			}
			if err := s.validateKey(tt.key); (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("validateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.wantKey, tt.key) {
				t.Errorf("validateKey() got = %v, want = %v", tt.key, tt.wantKey)
			}
		})
	}
}

func Test_sshHelperImpl_sshdCfgModified(t *testing.T) {
	log.Mute()
	sshdCfgFile := "/path/to/sshd_config"
	tests := []struct {
		name    string
		ev      *fsnotify.Event
		prepare func(w *MockfsWatcher, sysMgr *mocks.MocksysManager)
		want    bool
	}{
		{
			"return false if not an event related to the sshd_config",
			&fsnotify.Event{Name: "not/sshd_cfg/file"},
			nil,
			false,
		},
		{
			"return false if operation is not write, rename, or remove",
			&fsnotify.Event{
				Name: sshdCfgFile,
				Op:   ^(fsnotify.Write | fsnotify.Rename | fsnotify.Remove),
			},
			nil,
			false,
		},
		{
			"return true if is a Write operation",
			&fsnotify.Event{
				Name: sshdCfgFile,
				Op:   fsnotify.Write,
			},
			nil,
			true,
		},
		{
			"return true if contains a Write operation",
			&fsnotify.Event{
				Name: sshdCfgFile,
				Op:   fsnotify.Write | fsnotify.Create,
			},
			nil,
			true,
		},
		{
			"handle rename operation properly",
			&fsnotify.Event{
				Name: sshdCfgFile,
				Op:   fsnotify.Rename,
			},
			func(w *MockfsWatcher, sysMgr *mocks.MocksysManager) {
				gomock.InOrder(
					w.EXPECT().Remove(sshdCfgFile).Return(nil),
					sysMgr.EXPECT().FileExists(sshdCfgFile).Return(true, nil),
					w.EXPECT().Add(sshdCfgFile).Return(nil),
				)
			},
			true,
		},
		{
			"handle remove operation properly",
			&fsnotify.Event{
				Name: sshdCfgFile,
				Op:   fsnotify.Remove,
			},
			func(w *MockfsWatcher, sysMgr *mocks.MocksysManager) {
				gomock.InOrder(
					w.EXPECT().Remove(sshdCfgFile).Return(nil),
					sysMgr.EXPECT().FileExists(sshdCfgFile).Return(true, nil),
					w.EXPECT().Add(sshdCfgFile).Return(nil),
				)
			},
			true,
		},
		{
			"should ignore errors when removing the file hook",
			&fsnotify.Event{
				Name: sshdCfgFile,
				Op:   fsnotify.Rename,
			},
			func(w *MockfsWatcher, sysMgr *mocks.MocksysManager) {
				gomock.InOrder(
					w.EXPECT().Remove(sshdCfgFile).Return(errors.New("oops")),
					sysMgr.EXPECT().FileExists(sshdCfgFile).Return(true, nil),
					w.EXPECT().Add(sshdCfgFile).Return(nil),
				)
			},
			true,
		},
		{
			"should ignore errors when re-adding the file hook",
			&fsnotify.Event{
				Name: sshdCfgFile,
				Op:   fsnotify.Rename,
			},
			func(w *MockfsWatcher, sysMgr *mocks.MocksysManager) {
				gomock.InOrder(
					w.EXPECT().Remove(sshdCfgFile).Return(nil),
					sysMgr.EXPECT().FileExists(sshdCfgFile).Return(true, nil),
					w.EXPECT().Add(sshdCfgFile).Return(errors.New("oops")),
				)
			},
			true,
		},
		{
			"should retry with interval until sshd_config is back in place",
			&fsnotify.Event{
				Name: sshdCfgFile,
				Op:   fsnotify.Rename,
			},
			func(w *MockfsWatcher, sysMgr *mocks.MocksysManager) {
				gomock.InOrder(
					w.EXPECT().Remove(sshdCfgFile).Return(nil),
					sysMgr.EXPECT().FileExists(sshdCfgFile).Return(false, errors.New("oops")),
					sysMgr.EXPECT().Sleep(fileCheckInterval),
					sysMgr.EXPECT().FileExists(sshdCfgFile).Return(false, nil),
					sysMgr.EXPECT().Sleep(fileCheckInterval),
					sysMgr.EXPECT().FileExists(sshdCfgFile).Return(true, nil),
					w.EXPECT().Add(sshdCfgFile).Return(nil),
				)
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			sysMgrMock := mocks.NewMocksysManager(mockCtl)
			fsWatcherMock := NewMockfsWatcher(mockCtl)

			if tt.prepare != nil {
				tt.prepare(fsWatcherMock, sysMgrMock)
			}

			s := &sshHelperImpl{
				mgr: &SSHManager{
					sysMgr: sysMgrMock,
				},
			}
			if got := s.sshdCfgModified(fsWatcherMock, sshdCfgFile, tt.ev); got != tt.want {
				t.Errorf("sshdCfgModified() = %v, want %v", got, tt.want)
			}
		})
	}
}
