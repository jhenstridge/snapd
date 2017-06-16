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

package builtin

import (
	"fmt"
	"strings"

	"github.com/snapcore/snapd/interfaces"
	"github.com/snapcore/snapd/interfaces/apparmor"
	"github.com/snapcore/snapd/interfaces/mount"
	"github.com/snapcore/snapd/release"
)

const xdgDesktopPortalPermanentSlotAppArmor = `
# Description: Allow owning the Desktop portal bus names on the session bus

#include <abstractions/dbus-session-strict>

dbus (send)
    bus=session
    path=/org/freedesktop/DBus
    interface=org.freedesktop.DBus
    member=(RequestName|ReleaseName|GetConnectionCredentials|GetConnectionUnixProcessID)
    peer=(name=org.freedesktop.DBus, label=unconfined),

dbus (bind)
    bus=session
    name=org.freedesktop.portal.{Desktop,Documents},
`

const xdgDesktopPortalConnectedSlotAppArmor = `
# Description: allow client snaps to access the desktop portal service.
dbus (receive, send)
    bus=session
    interface=org.freedesktop.portal.*
    path=/org/freedesktop/portal/{desktop,documents}
    peer=(label=###PLUG_SECURITY_TAGS###),

dbus (receive, send)
    bus=session
    interface=org.freedesktop.DBus.Properties
    path=/org/freedesktop/portal/{desktop,documents}
    peer=(label=###PLUG_SECURITY_TAGS###),
`

const xdgDesktopPortalConnectedPlugAppArmor = `
# Description: allow access to the document portal file system.
owner /run/user/[0-9]*/doc/** rw,

# Description: allow access to the desktop portal D-Bus service.

#include <abstractions/dbus-session-strict>

dbus (receive, send)
    bus=session
    interface=org.freedesktop.portal.*
    path=/org/freedesktop/portal/{desktop,documents}
    peer=(label=###SLOT_SECURITY_TAGS###),

dbus (receive, send)
    bus=session
    interface=org.freedesktop.DBus.Properties
    path=/org/freedesktop/portal/{desktop,documents}
    peer=(label=###SLOT_SECURITY_TAGS###),
`

type XdgDesktopPortalInterface struct{}

func (iface *XdgDesktopPortalInterface) Name() string {
	return "xdg-desktop-portal"
}

func (iface *XdgDesktopPortalInterface) AppArmorConnectedPlug(spec *apparmor.Specification, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error {
	snippet := xdgDesktopPortalConnectedPlugAppArmor
	old := "###SLOT_SECURITY_TAGS###"
	var new string
	if release.OnClassic {
		// If we're running on classic, xdg-desktop-portal
		// will be running unconfined.
		new = "unconfined"
	} else {
		new = slotAppLabelExpr(slot)
	}
	snippet = strings.Replace(snippet, old, new, -1)
	spec.AddSnippet(snippet)
	return nil
}

func (iface *XdgDesktopPortalInterface) AppArmorPermanentSlot(spec *apparmor.Specification, slot *interfaces.Slot) error {
	spec.AddSnippet(xdgDesktopPortalPermanentSlotAppArmor)
	return nil
}

func (iface *XdgDesktopPortalInterface) AppArmorConnectedSlot(spec *apparmor.Specification, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error {
	snippet := xdgDesktopPortalConnectedSlotAppArmor
	old := "###PLUG_SNAP_NAME###"
	new := plug.Snap.Name()
	snippet = strings.Replace(snippet, old, new, -1)

	old = "###PLUG_SECURITY_TAGS###"
	new = plugAppLabelExpr(plug)
	snippet = strings.Replace(snippet, old, new, -1)
	spec.AddSnippet(snippet)
	return nil
}

func (iface *XdgDesktopPortalInterface) MountConnectedPlug(spec *mount.Specification, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error {
	appId := "snap.pkg." + plug.Snap.Name()
	// FIXME: this needs to be per-user.  As is, it only works for the first 
	spec.AddMountEntry(mount.Entry{
		Name: "/run/user/1000/doc/by-app/" + appId,
		Dir: "/run/user/1000/doc",
		Options: []string{"bind", "rw"},
	})
	return nil
}


func (iface *XdgDesktopPortalInterface) SanitizePlug(plug *interfaces.Plug) error {
	if iface.Name() != plug.Interface {
		panic(fmt.Sprintf("plug is not of interface %q", iface.Name()))
	}
	return nil
}

func (iface *XdgDesktopPortalInterface) SanitizeSlot(slot *interfaces.Slot) error {
	if iface.Name() != slot.Interface {
		panic(fmt.Sprintf("slot is not of interface %q", iface.Name()))
	}
	return nil
}

func (iface *XdgDesktopPortalInterface) AutoConnect(plug *interfaces.Plug, slot *interfaces.Slot) bool {
	return true
}


func init() {
	registerIface(&XdgDesktopPortalInterface{})
}

