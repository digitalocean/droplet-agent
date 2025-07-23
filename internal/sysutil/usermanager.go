// SPDX-License-Identifier: Apache-2.0

package sysutil

import (
	"os/user"
)

type userOperator interface {
	Current() (*user.User, error)
}

func newUserOperator() userOperator {
	return &userImpl{}
}

type userImpl struct{}

func (u *userImpl) Current() (*user.User, error) {
	return user.Current()
}
