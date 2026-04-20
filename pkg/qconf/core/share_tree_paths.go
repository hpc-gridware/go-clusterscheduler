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
	"strings"
)

// ErrShareTreeNodeNotFound is returned when a path does not resolve to any
// node in the share tree. Consumers can use errors.Is to distinguish
// missing-path from other failure modes.
var ErrShareTreeNodeNotFound = errors.New("share tree: node not found")

// shareTreeRootName is the canonical label for the share tree root. The
// qconf man page guarantees "Root" as the conventional name.
const shareTreeRootName = "Root"

// invalidSharePathChars are metacharacters that must not appear in any
// share-tree path or segment. They are rejected before the string is
// assembled into a qconf argv token because qconf accepts paths and
// `name=value` pairs separated by "," on the same argv element; a
// segment containing "," or "=" would forge additional name=value
// pairs (e.g. Node="/P1=999,/victim" turning `-mstnode` into a
// multi-node update the caller never authorised).
const invalidSharePathChars = ",=\n\r\t"

// ValidateSharePath checks that p is free of metacharacters that would
// let a caller inject additional node/value pairs into a qconf argv
// token. Whitespace-only input and the empty string are accepted here
// and left for NormalizeSharePath to reject with a clearer message.
//
// Callers: every CommandLineQConf method that accepts a caller-supplied
// share-tree path (Show/Add/Modify/Delete ShareTreeNodes,
// ShowShareTreeSubtree, etc.). NormalizeSharePath also runs it on the
// full input before segment extraction so structured-subtree ops get
// the same guard.
func ValidateSharePath(p string) error {
	if i := strings.IndexAny(p, invalidSharePathChars); i >= 0 {
		return fmt.Errorf("share tree: path contains invalid character %q at position %d",
			p[i], i)
	}
	return nil
}

// NormalizeSharePath converts a user-supplied path into the canonical
// "/Root/Seg1/Seg2" form. It accepts inputs with or without the leading
// slash and with or without the explicit "Root" prefix.
//
// Examples:
//
//	NormalizeSharePath("/Root/IT/devel") -> "/Root/IT/devel"
//	NormalizeSharePath("Root/IT/devel")  -> "/Root/IT/devel"
//	NormalizeSharePath("IT/devel")       -> "/Root/IT/devel"
//	NormalizeSharePath("/")              -> "/Root"
//	NormalizeSharePath("Root")           -> "/Root"
//
// Empty input (after trimming) returns an error. Paths containing
// metacharacters that would let a caller forge extra qconf name=value
// pairs (`,`, `=`, `\n`, `\r`, `\t`) are rejected via ValidateSharePath.
func NormalizeSharePath(p string) (string, error) {
	if err := ValidateSharePath(p); err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(p)
	// Strip leading and trailing slashes for segment extraction.
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		// "/" or "" is interpreted as the root path.
		if p == "/" {
			return "/" + shareTreeRootName, nil
		}
		return "", errors.New("share tree: empty path")
	}
	segments := strings.Split(trimmed, "/")
	// Reject empty segments (e.g. from "//IT//devel").
	for _, s := range segments {
		if s == "" {
			return "", errors.New("share tree: empty segment in path")
		}
	}
	if segments[0] != shareTreeRootName {
		segments = append([]string{shareTreeRootName}, segments...)
	}
	return "/" + strings.Join(segments, "/"), nil
}

// SplitSharePath breaks a canonical path into its segments. The returned
// slice always starts with "Root". Returns nil for malformed input.
//
// Example: SplitSharePath("/Root/IT/devel") -> ["Root", "IT", "devel"].
func SplitSharePath(p string) []string {
	normalized, err := NormalizeSharePath(p)
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimPrefix(normalized, "/"), "/")
}

// FindNodeByPath walks the tree rooted at root and returns the node
// matching the given path plus its parent. Parent is nil when the target
// is the root itself.
//
// Returns ErrShareTreeNodeNotFound when any segment fails to resolve or
// when root is nil. An invalid path (per NormalizeSharePath) is wrapped
// with the same sentinel.
func FindNodeByPath(root *StructuredShareTreeNode, path string) (node, parent *StructuredShareTreeNode, err error) {
	if root == nil {
		return nil, nil, ErrShareTreeNodeNotFound
	}
	segments := SplitSharePath(path)
	if len(segments) == 0 {
		return nil, nil, ErrShareTreeNodeNotFound
	}
	// The first segment must match the root's name.
	if segments[0] != root.Name {
		return nil, nil, ErrShareTreeNodeNotFound
	}
	current := root
	var currentParent *StructuredShareTreeNode
	for _, seg := range segments[1:] {
		var next *StructuredShareTreeNode
		for _, c := range current.Children {
			if c.Name == seg {
				next = c
				break
			}
		}
		if next == nil {
			return nil, nil, ErrShareTreeNodeNotFound
		}
		currentParent = current
		current = next
	}
	return current, currentParent, nil
}

// CloneShareTreeSubtree returns a deep copy of the subtree rooted at src.
// All descendant pointers are fresh; mutating the clone never affects the
// input. IDs are zeroed on the clone because FormatShareTreeText re-assigns
// them canonically on serialization.
//
// Returns nil when src is nil.
func CloneShareTreeSubtree(src *StructuredShareTreeNode) *StructuredShareTreeNode {
	if src == nil {
		return nil
	}
	clone := &StructuredShareTreeNode{
		ID:     0,
		Name:   src.Name,
		Type:   src.Type,
		Shares: src.Shares,
	}
	if len(src.Children) == 0 {
		return clone
	}
	clone.Children = make([]*StructuredShareTreeNode, len(src.Children))
	for i, c := range src.Children {
		clone.Children[i] = CloneShareTreeSubtree(c)
	}
	return clone
}

// IsDescendant reports whether candidate is descendant of ancestor, using
// pointer identity to walk. Returns false when either argument is nil.
// Used for cycle detection in MoveShareTreeSubtree.
func IsDescendant(ancestor, candidate *StructuredShareTreeNode) bool {
	if ancestor == nil || candidate == nil {
		return false
	}
	for _, c := range ancestor.Children {
		if c == candidate {
			return true
		}
		if IsDescendant(c, candidate) {
			return true
		}
	}
	return false
}
