/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2026 HPC-Gridware GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
************************************************************************/
/*___INFO__MARK_END__*/

package core

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ShareTreeNodeType is the qconf-level node kind flag. In the qconf -sstree
// text format, type=0 denotes a user (or interior organizational) node and
// type=1 denotes a project node.
type ShareTreeNodeType int

const (
	// ShareTreeNodeUser is a user or interior organizational node.
	ShareTreeNodeUser ShareTreeNodeType = 0
	// ShareTreeNodeProject is a project node.
	ShareTreeNodeProject ShareTreeNodeType = 1
)

// StructuredShareTreeNode is a fully typed view of a single node in the share
// tree. It carries every attribute the qconf text format emits (id, name,
// type, shares, childnodes) plus the Children slice, which is populated by
// ParseShareTreeText for ergonomic traversal.
type StructuredShareTreeNode struct {
	ID       int                        `json:"id"`
	Name     string                     `json:"name"`
	Type     ShareTreeNodeType          `json:"type"`
	Shares   int                        `json:"shares"`
	Children []*StructuredShareTreeNode `json:"children,omitempty"`
}

// StructuredShareTree is a rooted view of the entire share tree.
type StructuredShareTree struct {
	Root *StructuredShareTreeNode `json:"root"`
}

// ErrNoShareTree indicates that no share tree is currently configured.
// Consumers can use errors.Is to distinguish the empty-tree state from
// other failure modes.
var ErrNoShareTree = errors.New("no share tree defined")

// ParseShareTreeText parses the output of qconf -sstree into a structured
// tree. The format is documented in sge_share_tree(5).
//
// Returns (nil, ErrNoShareTree) when the input reports that no tree exists
// (empty input, or text containing "no sharetree" / "no share tree").
func ParseShareTreeText(raw string) (*StructuredShareTree, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, ErrNoShareTree
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "no sharetree") || strings.Contains(lower, "no share tree") {
		return nil, ErrNoShareTree
	}

	// The format is a sequence of attribute=value lines grouped into nodes.
	// A new node begins at each "id=" line. childnodes is either "NONE" or a
	// comma-separated list of child ids.
	nodesByID := make(map[int]*StructuredShareTreeNode)
	childIDsByID := make(map[int][]int)
	var order []int
	var current *StructuredShareTreeNode

	for _, rawLine := range strings.Split(raw, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])

		switch key {
		case "id":
			id, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("share tree: invalid id %q: %w", val, err)
			}
			if _, dup := nodesByID[id]; dup {
				return nil, fmt.Errorf("share tree: duplicate id %d", id)
			}
			node := &StructuredShareTreeNode{ID: id}
			nodesByID[id] = node
			order = append(order, id)
			current = node
		case "name":
			if current == nil {
				return nil, fmt.Errorf("share tree: name %q before id", val)
			}
			current.Name = val
		case "type":
			if current == nil {
				return nil, fmt.Errorf("share tree: type %q before id", val)
			}
			t, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("share tree: invalid type %q: %w", val, err)
			}
			current.Type = ShareTreeNodeType(t)
		case "shares":
			if current == nil {
				return nil, fmt.Errorf("share tree: shares %q before id", val)
			}
			s, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("share tree: invalid shares %q: %w", val, err)
			}
			current.Shares = s
		case "childnodes":
			if current == nil {
				return nil, fmt.Errorf("share tree: childnodes %q before id", val)
			}
			if strings.EqualFold(val, "NONE") {
				continue
			}
			for _, s := range strings.Split(val, ",") {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}
				cid, err := strconv.Atoi(s)
				if err != nil {
					return nil, fmt.Errorf("share tree: invalid child id %q: %w", s, err)
				}
				childIDsByID[current.ID] = append(childIDsByID[current.ID], cid)
			}
		}
	}

	if len(order) == 0 {
		return nil, ErrNoShareTree
	}

	// Wire up children by id. An unknown child id is a hard parse error.
	for id, childIDs := range childIDsByID {
		parent := nodesByID[id]
		for _, cid := range childIDs {
			child, ok := nodesByID[cid]
			if !ok {
				return nil, fmt.Errorf("share tree: node %d references unknown child %d", id, cid)
			}
			parent.Children = append(parent.Children, child)
		}
	}

	// The root is conventionally id=0; fall back to the first declared node
	// if id=0 is absent (defensive; the scheduler always emits id=0 first).
	root, ok := nodesByID[0]
	if !ok {
		root = nodesByID[order[0]]
	}
	return &StructuredShareTree{Root: root}, nil
}

// FormatShareTreeText serializes a structured tree into the text format
// consumed by qconf -Mstree. IDs are re-generated in pre-order starting at
// zero so the output is canonical regardless of the input node IDs.
//
// Returns an error when the tree or root is nil, or when any node in the
// tree is nil.
func FormatShareTreeText(t *StructuredShareTree) (string, error) {
	if t == nil || t.Root == nil {
		return "", errors.New("share tree: nil tree")
	}

	// Pre-order walk assigns canonical IDs.
	var flat []*StructuredShareTreeNode
	ids := make(map[*StructuredShareTreeNode]int)
	var walk func(n *StructuredShareTreeNode) error
	walk = func(n *StructuredShareTreeNode) error {
		if n == nil {
			return errors.New("share tree: nil node")
		}
		if _, seen := ids[n]; seen {
			return errors.New("share tree: cycle detected")
		}
		ids[n] = len(flat)
		flat = append(flat, n)
		for _, c := range n.Children {
			if err := walk(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(t.Root); err != nil {
		return "", err
	}

	var b strings.Builder
	for _, n := range flat {
		fmt.Fprintf(&b, "id=%d\n", ids[n])
		fmt.Fprintf(&b, "name=%s\n", n.Name)
		fmt.Fprintf(&b, "type=%d\n", int(n.Type))
		fmt.Fprintf(&b, "shares=%d\n", n.Shares)
		if len(n.Children) == 0 {
			b.WriteString("childnodes=NONE\n")
			continue
		}
		childIDs := make([]string, 0, len(n.Children))
		for _, c := range n.Children {
			childIDs = append(childIDs, strconv.Itoa(ids[c]))
		}
		fmt.Fprintf(&b, "childnodes=%s\n", strings.Join(childIDs, ","))
	}
	return b.String(), nil
}
