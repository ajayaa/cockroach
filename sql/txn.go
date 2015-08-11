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
// Author: Vivek Menezes (vivek@cockroachlabs.com)

package sql

import (
	"github.com/cockroachdb/cockroach/client"
	"github.com/cockroachdb/cockroach/proto"
	"github.com/cockroachdb/cockroach/sql/parser"
)

func (p *planner) BeginTransaction(n *parser.BeginTransaction) (planNode, error) {
	p.session.Txn = &proto.Transaction{}
	return &valuesNode{}, nil
}

func (p *planner) CommitTransaction(n *parser.CommitTransaction) (planNode, error) {
	var b client.Batch
	err := p.txn.Commit(&b)
	// Reset transaction.
	p.session.Txn = nil
	return &valuesNode{}, err
}

func (p *planner) RollbackTransaction(n *parser.RollbackTransaction) (planNode, error) {
	// TODO(vivek): Implement KV rollback transaction.
	p.session.Txn = nil
	return &valuesNode{}, nil
}
