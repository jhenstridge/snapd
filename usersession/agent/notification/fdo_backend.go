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
	"log"
	"sync"

	"github.com/godbus/dbus"
)

const (
	fdoBusName         = "org.freedesktop.Notifications"
	fdoObjectPath      = "/org/freedesktop/Notifications"
	fdoInterface       = "org.freedesktop.Notifications"
	fdoSignalMatchRule = "type='signal',sender='" + fdoBusName + "',path='" + fdoObjectPath + "',interface='" + fdoInterface + "'"
)

type fdoBackend struct {
	context   context.Context
	bus       *dbus.Conn
	manager   dbus.BusObject
	desktopID string

	mu            sync.Mutex
	notifications map[string]*fdoNotification
	idMap         map[uint32]string
}

type fdoNotification struct {
	fdoID        uint32
	notification Notification
}

func newFdoBackend(context context.Context, bus *dbus.Conn, desktopID string) (Manager, error) {
	b := &fdoBackend{
		context:       context,
		bus:           bus,
		manager:       bus.Object(fdoBusName, fdoObjectPath),
		desktopID:     desktopID,
		notifications: make(map[string]*fdoNotification),
		idMap:         make(map[uint32]string),
	}
	if err := bus.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, fdoSignalMatchRule).Store(); err != nil {
		return nil, err
	}
	go b.watchSignals()
	return b, nil
}

func (b *fdoBackend) watchSignals() {
	// Ideally we wouldn't need a buffer, but closing the channel is racy:
	// https://github.com/godbus/dbus/issues/158
	ch := make(chan *dbus.Signal, 10)
	b.bus.Signal(ch)
Loop:
	for {
		select {
		case <-b.context.Done():
			break Loop
		case s := <-ch:
			if s.Path != fdoObjectPath {
				continue Loop
			}
			if s.Name == fdoInterface+".NotificationClosed" {
				if len(s.Body) != 2 {
					log.Println("Received NotificationClosed signal with wrong number of arguments")
					continue Loop
				}
				var (
					id, reason uint32
					ok         bool
				)
				if id, ok = s.Body[0].(uint32); !ok {
					log.Println("First argument to NotificationClosed not a uint32")
					continue Loop
				}
				if reason, ok = s.Body[1].(uint32); !ok {
					log.Println("Second argument to NotificationClosed not a uint32")
					continue Loop
				}
				b.handleNotificationClosed(id, reason)
			} else if s.Name == fdoInterface+".ActionInvoked" {
				if len(s.Body) != 2 {
					log.Println("Received ActionInvoked signal with wrong number of arguments")
					continue Loop
				}
				var (
					id     uint32
					action string
					ok     bool
				)
				if id, ok = s.Body[0].(uint32); !ok {
					log.Println("First argument to ActionInvoked not a uint32")
					continue Loop
				}
				if action, ok = s.Body[1].(string); !ok {
					log.Println("Second argument to ActionInvoked not a string")
					continue Loop
				}
				b.handleActionInvoked(id, action)
			}
		}
	}
	b.bus.RemoveSignal(ch)
	close(ch)
	if err := b.bus.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, fdoSignalMatchRule).Store(); err != nil {
		log.Println("Can not remove signal watch: ", err)
	}
}

func (b *fdoBackend) handleNotificationClosed(fdoID, reason uint32) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if id, ok := b.idMap[fdoID]; ok {
		log.Println("NotificationClosed", id, reason)
		delete(b.notifications, id)
		delete(b.idMap, fdoID)
	}
}

func (b *fdoBackend) handleActionInvoked(fdoID uint32, fdoAction string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if id, ok := b.idMap[fdoID]; ok {
		info := b.notifications[id]
		// translate action:
		var action string
		if fdoAction == "default" {
			action = info.notification.DefaultAction
		} else {
			for _, button := range info.notification.Buttons {
				if fdoAction == button.Action {
					action = button.Action
					break
				}
			}
		}
		log.Println("ActionInvoked", id, action)
	}
}

func fdoUrgency(priority Priority) uint8 {
	switch priority {
	case PriorityLow:
		return 0
	case PriorityNormal, PriorityHigh:
		return 1
	case PriorityUrgent:
		return 2
	default:
		return 1 // default to normal
	}
}

func (b *fdoBackend) SendNotification(id string, notification Notification) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	info := b.notifications[id]
	if info == nil {
		info = &fdoNotification{}
	}

	actions := []string{}
	if notification.DefaultAction != "" {
		actions = append(actions, "default", "")
	}
	for _, button := range notification.Buttons {
		actions = append(actions, button.Action, button.Label)
	}
	hints := make(map[string]dbus.Variant)
	hints["desktop-entry"] = dbus.MakeVariant(b.desktopID)
	hints["urgency"] = dbus.MakeVariant(fdoUrgency(notification.Priority))
	if notification.Icon != "" {
		hints["icon-path"] = dbus.MakeVariant(notification.Icon)
	}

	var newID uint32
	if err := b.manager.Call(fdoInterface+".Notify", 0, "", info.fdoID, "", notification.Title, notification.Body, actions, hints, int32(-1)).Store(&newID); err != nil {
		return err
	}
	// Remove cached mapping of the old ID
	if info.fdoID != 0 {
		delete(b.idMap, info.fdoID)
	}
	info.fdoID = newID
	info.notification = notification
	b.idMap[newID] = id
	b.notifications[id] = info

	return nil
}

func (b *fdoBackend) WithdrawNotification(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	info := b.notifications[id]
	if info == nil {
		return nil
	}

	if err := b.manager.Call(fdoInterface+".CloseNotification", 0, info.fdoID).Store(); err != nil {
		return err
	}
	delete(b.idMap, info.fdoID)
	delete(b.notifications, id)
	return nil
}
