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

package sql

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroach/proto"
	"github.com/cockroachdb/cockroach/sql/parser"
	"github.com/cockroachdb/cockroach/structured"
	"github.com/cockroachdb/cockroach/util"
)

// Select selects rows from a single table. Select is the workhorse of the SQL
// statements. In the slowest and most general case, select must perform full
// table scans across multiple tables and sort and join the resulting rows on
// arbitrary columns. Full table scans can be avoided when indexes can be used
// to satisfy the where-clause.
//
// Privileges: READ on table
//   Notes: postgres requires SELECT. Also requires UPDATE on "FOR UPDATE".
//          mysql requires SELECT.
func (p *planner) Select(n *parser.Select) (planNode, error) {
	var desc *structured.TableDescriptor
	var index *structured.IndexDescriptor
	var visibleCols []structured.ColumnDescriptor

	switch len(n.From) {
	case 0:
		// desc remains nil.

	case 1:
		var err error
		desc, err = p.getAliasedTableDesc(n.From[0])
		if err != nil {
			return nil, err
		}

		if !desc.HasPrivilege(p.user, parser.PrivilegeRead) {
			return nil, fmt.Errorf("user %s does not have %s privilege on table %s",
				p.user, parser.PrivilegeRead, desc.Name)
		}

		// This is only kosher because we know that getAliasedDesc() succeeded.
		qname := n.From[0].(*parser.AliasedTableExpr).Expr.(*parser.QualifiedName)
		indexName := qname.Index()
		if indexName != "" && !strings.EqualFold(desc.PrimaryIndex.Name, indexName) {
			for i := range desc.Indexes {
				if strings.EqualFold(desc.Indexes[i].Name, indexName) {
					// Remove all but the matching index from the descriptor.
					desc.Indexes = desc.Indexes[i : i+1]
					index = &desc.Indexes[0]
					break
				}
			}
			if index == nil {
				return nil, fmt.Errorf("index \"%s\" not found", indexName)
			}
			// If the table was not aliased, use the index name instead of the table
			// name for fully-qualified columns in the expression.
			if n.From[0].(*parser.AliasedTableExpr).As == "" {
				desc.Alias = index.Name
			}
			// Strip out any columns from the table that are not present in the
			// index.
			indexColIDs := map[structured.ColumnID]struct{}{}
			for _, colID := range index.ColumnIDs {
				indexColIDs[colID] = struct{}{}
			}
			for _, col := range desc.Columns {
				if _, ok := indexColIDs[col.ID]; !ok {
					continue
				}
				visibleCols = append(visibleCols, col)
			}
		} else {
			index = &desc.PrimaryIndex
			visibleCols = desc.Columns
		}

	default:
		return nil, util.Errorf("TODO(pmattis): unsupported FROM: %s", n.From)
	}

	// Loop over the select expressions and expand them into the expressions
	// we're going to use to generate the returned column set and the names for
	// those columns.
	exprs := make([]parser.Expr, 0, len(n.Exprs))
	columns := make([]string, 0, len(n.Exprs))
	for _, e := range n.Exprs {
		// If a QualifiedName has a StarIndirection suffix we need to match the
		// prefix of the qualified name to one of the tables in the query and
		// then expand the "*" into a list of columns.
		if qname, ok := e.Expr.(*parser.QualifiedName); ok {
			if err := qname.NormalizeColumnName(); err != nil {
				return nil, err
			}
			if qname.IsStar() {
				if desc == nil {
					return nil, fmt.Errorf("\"%s\" with no tables specified is not valid", qname)
				}
				if e.As != "" {
					return nil, fmt.Errorf("\"%s\" cannot be aliased", qname)
				}
				tableName := qname.Table()
				if tableName != "" && !strings.EqualFold(desc.Alias, tableName) {
					return nil, fmt.Errorf("table \"%s\" not found", tableName)
				}

				if index != &desc.PrimaryIndex {
					for _, col := range index.ColumnNames {
						columns = append(columns, col)
						exprs = append(exprs, &parser.QualifiedName{Base: parser.Name(col)})
					}
				} else {
					for _, col := range desc.Columns {
						columns = append(columns, col.Name)
						exprs = append(exprs, &parser.QualifiedName{Base: parser.Name(col.Name)})
					}
				}
				continue
			}
		}

		exprs = append(exprs, e.Expr)
		if e.As != "" {
			columns = append(columns, string(e.As))
			continue
		}

		// If the expression is a qualified name, use the column name, not the
		// full qualification as the column name to return.
		switch t := e.Expr.(type) {
		case *parser.QualifiedName:
			if err := t.NormalizeColumnName(); err != nil {
				return nil, err
			}
			columns = append(columns, t.Column())
		default:
			columns = append(columns, e.Expr.String())
		}
	}

	// TODO(pmattis): Walk over the select exprs and WHERE clause looking for
	// simple cases where an index can be used. Start with the set of all indexes
	// and filter down the start and end keys based on the expressions in the
	// WHERE clause. Ensure that the columns referenced in the select expressions
	// are covered by the resulting indexes.

	s := &scanNode{
		txn:         p.txn,
		desc:        desc,
		index:       index,
		visibleCols: visibleCols,
		columns:     columns,
		render:      exprs,
	}
	if index != nil {
		s.isSecondaryIndex = index != &desc.PrimaryIndex
	}
	if n.Where != nil {
		s.filter = n.Where.Expr
	}

	if i, err := p.selectIndex(s); err != nil || i != nil {
		return i, err
	}
	return s, nil
}

type subqueryVisitor struct {
	*planner
	err error
}

var _ parser.Visitor = &subqueryVisitor{}

func (v *subqueryVisitor) Visit(expr parser.Expr) parser.Expr {
	if v.err != nil {
		return expr
	}
	subquery, ok := expr.(*parser.Subquery)
	if !ok {
		return expr
	}
	var plan planNode
	if plan, v.err = v.makePlan(subquery.Select); v.err != nil {
		return expr
	}
	var rows parser.DTuple
	for plan.Next() {
		values := plan.Values()
		switch len(values) {
		case 1:
			// TODO(pmattis): This seems hokey, but if we don't do this then the
			// subquery expands to a tuple of tuples instead of a tuple of values and
			// an expression like "k IN (SELECT foo FROM bar)" will fail because
			// we're comparing a single value against a tuple. Perhaps comparison of
			// a single value against a tuple should succeed if the tuple is one
			// element in length.
			rows = append(rows, values[0])
		default:
			// The result from plan.Values() is only valid until the next call to
			// plan.Next(), so make a copy.
			valuesCopy := make(parser.DTuple, len(values))
			copy(valuesCopy, values)
			rows = append(rows, valuesCopy)
		}
	}
	v.err = plan.Err()
	return rows
}

func (p *planner) expandSubqueries(stmt parser.Statement) error {
	v := subqueryVisitor{planner: p}
	parser.WalkStmt(&v, stmt)
	return v.err
}

func (p *planner) selectIndex(s *scanNode) (planNode, error) {
	if s.desc == nil {
		return nil, nil
	}

	indexes := make([]indexState, len(s.desc.Indexes)+1)
	indexes[0].desc = s.desc
	indexes[0].index = s.desc.PrimaryIndex
	for i := range s.desc.Indexes {
		indexes[i+1].desc = s.desc
		indexes[i+1].index = s.desc.Indexes[i]
	}

	for _, index := range indexes {
		index.score(s)
	}

	return nil, nil
}

type indexState struct {
	desc     *structured.TableDescriptor
	index    structured.IndexDescriptor
	beginKey proto.Key
	endKey   proto.Key
}

var _ parser.Visitor = &indexState{}

func (v *indexState) Visit(expr parser.Expr) parser.Expr {
	return expr
}

func (v *indexState) score(s *scanNode) {
	v.beginKey = proto.Key(structured.MakeIndexKeyPrefix(v.desc.ID, v.index.ID))
	v.endKey = v.beginKey.PrefixEnd()
	parser.WalkExpr(v, s.filter)
}
