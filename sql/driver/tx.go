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
// Author: Peter Mattis (peter@cockroachlabs.com)

package driver

import "database/sql/driver"

type tx struct {
	conn *conn
}

func (t *tx) Commit() error {
	if _, err := t.Exec("COMMIT TRANSACTION"); err != nil {
		return err
	}
	return nil
}

func (t *tx) Rollback() error {
	if _, err := t.Exec("ROLLBACK TRANSACTION"); err != nil {
		return err
	}
	return nil
}

func (t *tx) Exec(query string, args ...interface{}) (driver.Result, error) {
	var values []driver.Value
	for _, arg := range args {
		values = append(values, arg)
	}
	return t.conn.Exec(query, values)
}

func (t *tx) Prepare(query string) (*driver.Stmt, error) {
	stmt, err := t.conn.Prepare(query)
	return &stmt, err
}

func (t *tx) Query(query string, args ...interface{}) (r *rows, err error) {
	var values []driver.Value
	for _, arg := range args {
		values = append(values, arg)
	}
	return t.conn.Query(query, values)
}
