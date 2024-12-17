// SPDX-License-Identifier: Apache-2.0

package actioner

import (
	"testing"

	"github.com/digitalocean/droplet-agent/internal/metadata"
	"github.com/digitalocean/droplet-agent/internal/reservedipv6"
	"go.uber.org/mock/gomock"
)

func TestReservedIPv6Actioner_Do(t *testing.T) {
	t.Run("assign", func(t *testing.T) {
		// Arrange
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		rip6Manager := reservedipv6.NewMockManager(ctrl)
		actioner := NewReservedIPv6Actioner(rip6Manager)

		rip6Manager.EXPECT().Assign("2001:4860:4860::8888").Return(nil)

		// Act
		actioner.Do(&metadata.Metadata{
			ReservedIP: &metadata.ReservedIP{
				IPv6: &metadata.ReservedIPv6{
					Active:    true,
					IPAddress: "2001:4860:4860::8888",
				},
			},
		})
	})

	t.Run("unassign", func(t *testing.T) {
		// Arrange
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		rip6Manager := reservedipv6.NewMockManager(ctrl)
		actioner := NewReservedIPv6Actioner(rip6Manager)

		rip6Manager.EXPECT().Unassign().Return(nil)

		// Act
		actioner.Do(&metadata.Metadata{
			ReservedIP: &metadata.ReservedIP{
				IPv6: &metadata.ReservedIPv6{
					Active: false,
				},
			},
		})
	})
}
