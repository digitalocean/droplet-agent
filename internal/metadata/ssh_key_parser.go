package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/digitalocean/droplet-agent/internal/sysaccess"
	"regexp"
	"strings"
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

// FromPublicKey parses a string public key and attempts to convert it to a SSHKey object
func (p *SSHKeyParser) FromPublicKey(key string) (*sysaccess.SSHKey, error) {
	ret := &sysaccess.SSHKey{
		PublicKey: strings.Trim(key, " \t\r\n"),
		Type:      sysaccess.SSHKeyTypeDroplet,
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
	if err := json.Unmarshal([]byte(key), ret); err != nil {
		return nil, fmt.Errorf("%w:invalid key", err)
	}
	return ret, nil
}
