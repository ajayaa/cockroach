// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.
//
// Author: Marc Berhault (marc@cockroachlabs.com)

package privilege

import (
	"sort"
	"strings"
)

// Kind defines a privilege. This is output by the parser,
// and used to generate the privilege bitfields in the PrivilegeDescriptor.
type Kind uint32

// List of privileges. ALL is specifically encoded so that it will automatically
// pick up new privileges.
// TODO(marc): deprecate READ|WRITE.
const (
	_        = iota
	ALL Kind = iota
	CREATE
	DROP
	GRANT
	SELECT
	INSERT
	DELETE
	UPDATE
	READ
	WRITE
)

// nameMap maps privilege kinds to their names.
var nameMap = map[Kind]string{
	ALL:    "ALL",
	CREATE: "CREATE",
	DROP:   "DROP",
	GRANT:  "GRANT",
	SELECT: "SELECT",
	INSERT: "INSERT",
	DELETE: "DELETE",
	UPDATE: "UPDATE",
	READ:   "READ",
	WRITE:  "WRITE",
}

// ByValue is just an array of privilege kinds sorted by value.
var ByValue = [...]Kind{
	ALL, CREATE, DROP, GRANT, SELECT, INSERT, DELETE, UPDATE, READ, WRITE,
}

// String implements the Stringer interface for Privilege.
func (p Kind) String() string {
	return nameMap[p]
}

// List is a list of privileges.
type List []Kind

// Len, Swap, and Less implement the Sort interface.
func (pl List) Len() int {
	return len(pl)
}

func (pl List) Swap(i, j int) {
	pl[i], pl[j] = pl[j], pl[i]
}

func (pl List) Less(i, j int) bool {
	return pl[i] < pl[j]
}

// String implements the Stringer interface.
// This keeps the existing order and uses ", " as separator.
func (pl List) String() string {
	ret := make([]string, len(pl), len(pl))
	for i, p := range pl {
		ret[i] = p.String()
	}
	return strings.Join(ret, ", ")
}

// SortedString is similar to String() but returns
// privileges sorted by name and uses "," as separator.
func (pl List) SortedString() string {
	ret := make([]string, len(pl), len(pl))
	for i, p := range pl {
		ret[i] = p.String()
	}
	sort.Strings(ret)
	return strings.Join(ret, ",")
}
