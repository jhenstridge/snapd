// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package builtin_test

import (
	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/interfaces"
	"github.com/snapcore/snapd/interfaces/apparmor"
	"github.com/snapcore/snapd/interfaces/mount"
	"github.com/snapcore/snapd/interfaces/builtin"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/snap/snaptest"
	"github.com/snapcore/snapd/testutil"
)

type XdgDesktopPortalInterfaceSuite struct {
	iface       interfaces.Interface
	coreSlot    *interfaces.Slot
	classicSlot *interfaces.Slot
	plug        *interfaces.Plug
}

var _ = Suite(&XdgDesktopPortalInterfaceSuite{})

func (s *XdgDesktopPortalInterfaceSuite) SetUpTest(c *C) {
	// a desktop portal slot on a desktop portal snap
	const xdgDesktopPortalMockCoreSlotSnapInfoYaml = `name: xdg-desktop-portal
version: 1.0
apps:
 app:
  command: foo
  slots: [xdg-desktop-portal]
`
	// a xdg-desktop-portal slot on the core snap (as automatically added on classic)
	const xdgDesktopPortalMockClassicSlotSnapInfoYaml = `name: core
type: os
slots:
 xdg-desktop-portal:
  interface: xdg-desktop-portal
`
	const mockPlugSnapInfo = `name: client
version: 1.0
apps:
 app:
  command: foo
  plugs: [xdg-desktop-portal]
`
	s.iface = &builtin.XdgDesktopPortalInterface{}
	// xdg-desktop-portal snap with xdg-desktop-portal slot on an core/all-snap install.
	snapInfo := snaptest.MockInfo(c, xdgDesktopPortalMockCoreSlotSnapInfoYaml, nil)
	s.coreSlot = &interfaces.Slot{SlotInfo: snapInfo.Slots["xdg-desktop-portal"]}
	// xdg-desktop-portal slot on a core snap in a classic install.
	snapInfo = snaptest.MockInfo(c, xdgDesktopPortalMockClassicSlotSnapInfoYaml, nil)
	s.classicSlot = &interfaces.Slot{SlotInfo: snapInfo.Slots["xdg-desktop-portal"]}

	plugSnap := snaptest.MockInfo(c, mockPlugSnapInfo, nil)
	s.plug = &interfaces.Plug{PlugInfo: plugSnap.Plugs["xdg-desktop-portal"]}
}

func (s *XdgDesktopPortalInterfaceSuite) TestName(c *C) {
	c.Check(s.iface.Name(), Equals, "xdg-desktop-portal")
}

func (s *XdgDesktopPortalInterfaceSuite) TestSanitizeIncorrectInterface(c *C) {
	c.Check(func() { s.iface.SanitizeSlot(&interfaces.Slot{SlotInfo: &snap.SlotInfo{Interface: "other"}}) },
		PanicMatches, `slot is not of interface "xdg-desktop-portal"`)
	c.Check(func() { s.iface.SanitizePlug(&interfaces.Plug{PlugInfo: &snap.PlugInfo{Interface: "other"}}) },
		PanicMatches, `plug is not of interface "xdg-desktop-portal"`)
}

func (s *XdgDesktopPortalInterfaceSuite) TestUsedSecuritySystems(c *C) {
	// connected slots have a non-nil security snippet for apparmor
	apparmorSpec := &apparmor.Specification{}
	err := apparmorSpec.AddConnectedSlot(s.iface, s.plug, s.coreSlot)
	c.Assert(err, IsNil)
	c.Check(apparmorSpec.SecurityTags(), DeepEquals, []string{"snap.xdg-desktop-portal.app"})
	c.Check(apparmorSpec.SnippetForTag("snap.xdg-desktop-portal.app"), testutil.Contains, `path=/org/freedesktop/portal/{desktop,documents}`)

	// slots have no permanent snippet on classic
	apparmorSpec = &apparmor.Specification{}
	err = apparmorSpec.AddConnectedSlot(s.iface, s.plug, s.classicSlot)
	c.Assert(err, IsNil)
	c.Check(apparmorSpec.SecurityTags(), HasLen, 0)

	// slots have a permanent non-nil security snippet for apparmor
	apparmorSpec = &apparmor.Specification{}
	err = apparmorSpec.AddPermanentSlot(s.iface, s.coreSlot)
	c.Assert(err, IsNil)
	c.Check(apparmorSpec.SecurityTags(), DeepEquals, []string{"snap.xdg-desktop-portal.app"})
	c.Check(apparmorSpec.SnippetForTag("snap.xdg-desktop-portal.app"), testutil.Contains, `name=org.freedesktop.portal.{Desktop,Documents}`)

	// plugs have no permanent snippet
	apparmorSpec = &apparmor.Specification{}
	err = apparmorSpec.AddPermanentPlug(s.iface, s.plug)
	c.Assert(err, IsNil)
	c.Check(apparmorSpec.SecurityTags(), HasLen, 0)

	// connected plugs have a non-nil security snippet for apparmor
	apparmorSpec = &apparmor.Specification{}
	err = apparmorSpec.AddConnectedPlug(s.iface, s.plug, s.coreSlot)
	c.Assert(err, IsNil)
	c.Check(apparmorSpec.SecurityTags(), DeepEquals, []string{"snap.client.app"})
	c.Check(apparmorSpec.SnippetForTag("snap.client.app"), testutil.Contains, `path=/org/freedesktop/portal/{desktop,documents}`)

	// connected plugs have a bind mount of the document portal
	mountSpec := &mount.Specification{}
	err = mountSpec.AddConnectedPlug(s.iface, s.plug, s.classicSlot)
	c.Assert(err, IsNil)
	mounts := mountSpec.MountEntries()
	c.Assert(mounts, HasLen, 1)
	c.Check(mounts[0].Name, Equals, "/run/user/1000/doc/by-app/snap.pkg.client")
	c.Check(mounts[0].Dir, Equals, "/run/user/1000/doc")
	c.Check(mounts[0].Options, DeepEquals, []string{"bind"})
}
