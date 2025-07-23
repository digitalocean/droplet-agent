// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/digitalocean/droplet-agent/internal/sysaccess"
)

const (
	osUserRegex = `-os_user=(\S+)`
)

// SSHKeyParser is used for parsing different type of SSH Keys from metadata
type SSHKeyParser struct {
	pubKeyOSUserRegex *regexp.Regexp
}

// NewSSHKeyParser returns a new SSH Key parser
func NewSSHKeyParser() *SSHKeyParser {
	return &SSHKeyParser{
		pubKeyOSUserRegex: regexp.MustCompile(osUserRegex),
	}
}

func containsNewlineOrEncodedNewline(s string) bool {
	return strings.Contains(s, "\n") ||
		strings.Contains(s, "\r") ||
		strings.Contains(s, "%0A") ||
		strings.Contains(s, "%0D") ||
		strings.Contains(s, "%0a") ||
		strings.Contains(s, "%0d")
}

// FromPublicKey parses a string public key and attempts to convert it to a SSHKey object
func (p *SSHKeyParser) FromPublicKey(key string) (*sysaccess.SSHKey, error) {
	ret := &sysaccess.SSHKey{
		PublicKey: strings.Trim(key, " \t\r\n"),
		Type:      sysaccess.SSHKeyTypeDroplet,
	}
	if containsNewlineOrEncodedNewline(ret.PublicKey) {
		return nil, errors.New("invalid public key: contains newline or encoded newline characters")
	}
	match := p.pubKeyOSUserRegex.FindAllStringSubmatch(key, -1)
	if len(match) != 0 {
		// trying to find "-os_user" flag from the key
		// if not presented, default user will be used
		// if multiple os_user flags presented, the last one will be used because the backend service always append the
		// os_user to the end of the key
		lastIdx := len(match) - 1
		if len(match[lastIdx]) != 2 {
			// this should never happen, but just in case
			return nil, errors.New("invalid os_user")
		}
		ret.OSUser = match[lastIdx][1]
	}
	return ret, nil
}

// FromDOTTYKey parses a string dotty key and converts it into a SSHKey object
func (p *SSHKeyParser) FromDOTTYKey(key string) (*sysaccess.SSHKey, error) {
	ret := &sysaccess.SSHKey{
		Type: sysaccess.SSHKeyTypeDOTTY,
	}
	if err := json.Unmarshal([]byte(strings.Trim(key, " \t\r\n")), ret); err != nil {
		return nil, fmt.Errorf("%w:invalid key", err)
	}
	if containsNewlineOrEncodedNewline(ret.PublicKey) {
		return nil, errors.New("invalid public key: contains newline or encoded newline characters")
	}
	return ret, nil
}
