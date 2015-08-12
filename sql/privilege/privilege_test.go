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

package privilege_test

import (
	"testing"

	"github.com/cockroachdb/cockroach/sql/privilege"
	"github.com/cockroachdb/cockroach/util/leaktest"
)

func TestPrivilegeString(t *testing.T) {
	defer leaktest.AfterTest(t)
	testCases := []struct {
		privList         privilege.List
		stringer, sorted string
	}{
		{privilege.List{}, "", ""},
		// We avoid 0 as a privilege value even though we use 1 << privValue.
		{privilege.List{privilege.ALL}, "ALL", "ALL"},
		{privilege.List{privilege.ALL, privilege.DROP}, "ALL, DROP", "ALL,DROP"},
		{privilege.List{privilege.GRANT, privilege.DELETE}, "GRANT, DELETE", "DELETE,GRANT"},
		{privilege.List{privilege.ALL, privilege.CREATE, privilege.DROP, privilege.GRANT,
			privilege.SELECT, privilege.INSERT, privilege.DELETE, privilege.UPDATE,
			privilege.READ, privilege.WRITE},
			"ALL, CREATE, DROP, GRANT, SELECT, INSERT, DELETE, UPDATE, READ, WRITE",
			"ALL,CREATE,DELETE,DROP,GRANT,INSERT,READ,SELECT,UPDATE,WRITE",
		},
		{privilege.List{privilege.DROP, privilege.ALL}, "DROP, ALL", "ALL,DROP"},
		{privilege.List{privilege.DELETE, privilege.GRANT, privilege.DROP, privilege.INSERT},
			"DELETE, GRANT, DROP, INSERT", "DELETE,DROP,GRANT,INSERT"},
	}

	for _, tc := range testCases {
		pl := tc.privList
		if pl.String() != tc.stringer {
			t.Fatalf("%+v: wrong String() output: %q", tc, pl.String())
		}
		if pl.SortedString() != tc.sorted {
			t.Fatalf("%+v: wrong SortedString() output: %q", tc, pl.SortedString())
		}
	}
}
