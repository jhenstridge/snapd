// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2019 Canonical Ltd
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

package notification

import (
	"context"

	"github.com/godbus/dbus"
)

const (
	gtkBusName    = "org.gtk.Notifications"
	gtkObjectPath = "/org/gtk/Notifications"
	gtkInterface  = "org.gtk.Notifications"
)

type gtkBackend struct {
	context   context.Context
	bus       *dbus.Conn
	manager   dbus.BusObject
	desktopID string
}

func newGtkBackend(context context.Context, bus *dbus.Conn, desktopID string) (Manager, error) {
	// If the D-Bus service is not already running, assume it is
	// not available.
	var owner string
	if err := bus.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, gtkBusName).Store(&owner); err != nil {
		return nil, err
	}

	b := &gtkBackend{
		context:   context,
		bus:       bus,
		manager:   bus.Object(gtkBusName, gtkObjectPath),
		desktopID: desktopID,
	}
	return b, nil
}

func gtkPriority(priority Priority) string {
	switch priority {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityUrgent:
		return "urgent"
	default:
		return "normal" // default to normal
	}
}

func (b *gtkBackend) SendNotification(id string, notification Notification) error {
	info := make(map[string]dbus.Variant)

	if notification.Title != "" {
		info["title"] = dbus.MakeVariant(notification.Title)
	}
	if notification.Body != "" {
		info["body"] = dbus.MakeVariant(notification.Body)
	}
	if notification.Icon != "" {
		// todo: check g_icon_serialize
		info["icon"] = dbus.MakeVariant(notification.Icon)
	}
	info["priority"] = dbus.MakeVariant(gtkPriority(notification.Priority))
	if notification.DefaultAction != "" {
		info["default-action"] = dbus.MakeVariant(notification.DefaultAction)
	}
	var buttons []map[string]dbus.Variant
	for _, button := range notification.Buttons {
		buttonInfo := map[string]dbus.Variant{
			"label":  dbus.MakeVariant(button.Label),
			"action": dbus.MakeVariant(button.Action),
		}
		buttons = append(buttons, buttonInfo)
	}
	if len(buttons) != 0 {
		info["buttons"] = dbus.MakeVariant(buttons)
	}

	return b.manager.Call(gtkInterface+".AddNotification", 0, b.desktopID, id, info).Store()
}

func (b *gtkBackend) WithdrawNotification(id string) error {
	return b.manager.Call(gtkInterface+".RemoveNotification", 0, b.desktopID, id).Store()
}
