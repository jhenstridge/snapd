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

type Manager interface {
	SendNotification(id string, notification Notification) error
	WithdrawNotification(id string) error
}

func New(context context.Context, desktopID string) (Manager, error) {
	bus, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}

	// First try the GTK backend
	//if manager, err := newGtkBackend(context, bus, desktopID); err == nil {
	//	return manager, nil
	//}

	// Fallback to the older FDO API
	return newFdoBackend(context, bus, desktopID)
}
