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

// Notification represents a desktop notification
type Notification struct {
	Title         string
	Body          string
	Icon          string
	Priority      Priority
	Buttons       []Button
	DefaultAction string
}

type Priority uint32

const (
	PriorityNormal Priority = iota
	PriorityLow
	PriorityHigh
	PriorityUrgent
)

type Button struct {
	Label  string
	Action string
}
