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

package structured

import (
	"testing"

	"github.com/cockroachdb/cockroach/sql/privilege"
	"github.com/cockroachdb/cockroach/util/leaktest"
)

func TestPrivilegeDecode(t *testing.T) {
	defer leaktest.AfterTest(t)
	testCases := []struct {
		raw              uint32
		privileges       privilege.List
		stringer, sorted string
	}{
		{0, privilege.List{}, "", ""},
		// We avoid 0 as a privilege value even though we use 1 << privValue.
		{1, privilege.List{}, "", ""},
		{2, privilege.List{privilege.ALL}, "ALL", "ALL"},
		{10, privilege.List{privilege.ALL, privilege.DROP}, "ALL, DROP", "ALL,DROP"},
		{144, privilege.List{privilege.GRANT, privilege.DELETE}, "GRANT, DELETE", "DELETE,GRANT"},
		{2047,
			privilege.List{privilege.ALL, privilege.CREATE, privilege.DROP, privilege.GRANT,
				privilege.SELECT, privilege.INSERT, privilege.DELETE, privilege.UPDATE,
				privilege.READ, privilege.WRITE},
			"ALL, CREATE, DROP, GRANT, SELECT, INSERT, DELETE, UPDATE, READ, WRITE",
			"ALL,CREATE,DELETE,DROP,GRANT,INSERT,READ,SELECT,UPDATE,WRITE",
		},
	}

	for _, tc := range testCases {
		pl := rawToPrivilegeList(tc.raw)
		if len(pl) != len(tc.privileges) {
			t.Fatalf("%+v: wrong privilege list from raw: %+v", tc, pl)
		}
		for i := 0; i < len(pl); i++ {
			if pl[i] != tc.privileges[i] {
				t.Fatalf("%+v: wrong privilege list from raw: %+v", tc, pl)
			}
		}
		if pl.String() != tc.stringer {
			t.Fatalf("%+v: wrong String() output: %q", tc, pl.String())
		}
		if pl.SortedString() != tc.sorted {
			t.Fatalf("%+v: wrong SortedString() output: %q", tc, pl.SortedString())
		}
	}
}

func TestPrivilege(t *testing.T) {
	defer leaktest.AfterTest(t)
	descriptor := NewDefaultDatabasePrivilegeDescriptor()

	testCases := []struct {
		grantee       string // User to grant/revoke privileges on.
		grant, revoke privilege.List
		show          []UserPrivilegeString
	}{
		{"", nil, nil,
			[]UserPrivilegeString{{"root", "ALL"}},
		},
		{"root", privilege.List{privilege.ALL}, nil,
			[]UserPrivilegeString{{"root", "ALL"}},
		},
		{"root", privilege.List{privilege.INSERT, privilege.DROP}, nil,
			[]UserPrivilegeString{{"root", "ALL"}},
		},
		{"foo", privilege.List{privilege.INSERT, privilege.DROP}, nil,
			[]UserPrivilegeString{{"foo", "DROP,INSERT"}, {"root", "ALL"}},
		},
		{"bar", nil, privilege.List{privilege.INSERT, privilege.ALL},
			[]UserPrivilegeString{{"foo", "DROP,INSERT"}, {"root", "ALL"}},
		},
		{"foo", privilege.List{privilege.ALL}, nil,
			[]UserPrivilegeString{{"foo", "ALL"}, {"root", "ALL"}},
		},
		{"foo", nil, privilege.List{privilege.SELECT, privilege.INSERT, privilege.READ, privilege.WRITE},
			[]UserPrivilegeString{{"foo", "CREATE,DELETE,DROP,GRANT,UPDATE"}, {"root", "ALL"}},
		},
		{"foo", nil, privilege.List{privilege.ALL},
			[]UserPrivilegeString{{"root", "ALL"}},
		},
		// Validate checks that root still has ALL privileges, but we do not call it here.
		{"root", nil, privilege.List{privilege.ALL},
			[]UserPrivilegeString{},
		},
	}

	for tcNum, tc := range testCases {
		if tc.grantee != "" {
			if tc.grant != nil {
				descriptor.Grant(tc.grantee, tc.grant)
			}
			if tc.revoke != nil {
				descriptor.Revoke(tc.grantee, tc.revoke)
			}
		}
		show, err := descriptor.Show()
		if err != nil {
			t.Fatal(err)
		}
		if len(show) != len(tc.show) {
			t.Fatalf("#%d: show output for descriptor %+v differs, got: %+v, expected %+v",
				tcNum, descriptor, show, tc.show)
		}
		for i := 0; i < len(show); i++ {
			if show[i].User != tc.show[i].User || show[i].Privileges != tc.show[i].Privileges {
				t.Fatalf("#%d: show output for descriptor %+v differs, got: %+v, expected %+v",
					tcNum, descriptor, show, tc.show)
			}
		}
	}
}

func TestPrivilegeValidate(t *testing.T) {
	defer leaktest.AfterTest(t)
	descriptor := NewDefaultDatabasePrivilegeDescriptor()
	if err := descriptor.Validate(); err != nil {
		t.Fatal(err)
	}
	descriptor.Grant("foo", privilege.List{privilege.ALL})
	if err := descriptor.Validate(); err != nil {
		t.Fatal(err)
	}
	descriptor.Grant("root", privilege.List{privilege.SELECT})
	if err := descriptor.Validate(); err != nil {
		t.Fatal(err)
	}
	descriptor.Revoke("root", privilege.List{privilege.SELECT})
	if err := descriptor.Validate(); err == nil {
		t.Fatal("unexpected success")
	}
	// TODO(marc): validate fails here because we do not aggregate
	// privileges into ALL when all are set.
	descriptor.Grant("root", privilege.List{privilege.SELECT})
	if err := descriptor.Validate(); err == nil {
		t.Fatal("unexpected success")
	}
	descriptor.Revoke("root", privilege.List{privilege.ALL})
	if err := descriptor.Validate(); err == nil {
		t.Fatal("unexpected success")
	}
}
