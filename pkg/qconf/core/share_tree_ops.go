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
	"fmt"
)

// The functions in this file are pure: they take a tree and an operation
// description and return a new tree (or an error) without performing any
// I/O. The QConf CommandLine implementations in qconf_impl.go are thin
// wrappers that read the current tree, call one of these, validate, and
// write back via qconf -Mstree.
//
// Inputs are never mutated. Every operation deep-clones the input tree
// before touching it so callers can safely discard results on error.

// pathNotFound builds a single-element ShareTreeValidationErrors signaling
// a missing path. Called from 14 sites across this file, which is why it
// stays extracted rather than inlined: a literal ShareTreeValidationErrors
// at every call site would add ~70 lines of noise without adding clarity.
func pathNotFound(path string) *ShareTreeValidationErrors {
	return &ShareTreeValidationErrors{Errs: []ShareTreeValidationError{{
		Code:    ShareCodePathNotFound,
		Path:    path,
		Message: fmt.Sprintf("no node at path %s", path),
	}}}
}

// ApplySubtreeReplace returns a new tree in which the subtree at path is
// replaced by sub. The returned tree has been validated.
//
// If path resolves to Root, the entire tree is replaced (subject to
// validation). Otherwise the sibling ordering at the parent is preserved.
func ApplySubtreeReplace(t *StructuredShareTree, path string, sub *StructuredShareTreeNode, opts *ShareTreeValidationOptions) (*StructuredShareTree, error) {
	if t == nil || t.Root == nil {
		return nil, ErrNoShareTree
	}
	if sub == nil {
		return nil, fmt.Errorf("share tree: replacement subtree is nil")
	}

	normalized, err := NormalizeSharePath(path)
	if err != nil {
		return nil, pathNotFound(path)
	}

	dup := &StructuredShareTree{Root: CloneShareTreeSubtree(t.Root)}
	target, parent, err := FindNodeByPath(dup.Root, normalized)
	if err != nil {
		return nil, pathNotFound(normalized)
	}

	replacement := CloneShareTreeSubtree(sub)
	if parent == nil {
		dup.Root = replacement
	} else {
		for i, c := range parent.Children {
			if c == target {
				parent.Children[i] = replacement
				break
			}
		}
	}
	if errs := ValidateShareTree(dup, opts); len(errs) > 0 {
		return nil, &ShareTreeValidationErrors{Errs: errs}
	}
	return dup, nil
}

// ApplySubtreeAdd returns a new tree with sub inserted as a new child of
// parentPath. The returned tree has been validated.
func ApplySubtreeAdd(t *StructuredShareTree, parentPath string, sub *StructuredShareTreeNode, opts *ShareTreeValidationOptions) (*StructuredShareTree, error) {
	if t == nil || t.Root == nil {
		return nil, ErrNoShareTree
	}
	if sub == nil {
		return nil, fmt.Errorf("share tree: added subtree is nil")
	}

	normalized, err := NormalizeSharePath(parentPath)
	if err != nil {
		return nil, pathNotFound(parentPath)
	}

	dup := &StructuredShareTree{Root: CloneShareTreeSubtree(t.Root)}
	parentNode, _, err := FindNodeByPath(dup.Root, normalized)
	if err != nil {
		return nil, pathNotFound(normalized)
	}

	addition := CloneShareTreeSubtree(sub)
	parentNode.Children = append(parentNode.Children, addition)
	if errs := ValidateShareTree(dup, opts); len(errs) > 0 {
		return nil, &ShareTreeValidationErrors{Errs: errs}
	}
	return dup, nil
}

// ApplySubtreeDelete returns a new tree with the subtree at path removed.
// Refuses to delete the root via this helper; callers that want to drop
// the whole tree should use the tree-level delete API (qconf -dstree).
func ApplySubtreeDelete(t *StructuredShareTree, path string) (*StructuredShareTree, error) {
	if t == nil || t.Root == nil {
		return nil, ErrNoShareTree
	}

	normalized, err := NormalizeSharePath(path)
	if err != nil {
		return nil, pathNotFound(path)
	}

	dup := &StructuredShareTree{Root: CloneShareTreeSubtree(t.Root)}
	target, parent, err := FindNodeByPath(dup.Root, normalized)
	if err != nil {
		return nil, pathNotFound(normalized)
	}
	if parent == nil {
		return nil, &ShareTreeValidationErrors{Errs: []ShareTreeValidationError{{
			Code:    ShareCodeRootDelete,
			Path:    normalized,
			Message: "cannot delete the root via subtree delete; use the tree-level delete endpoint",
		}}}
	}

	newChildren := make([]*StructuredShareTreeNode, 0, len(parent.Children)-1)
	for _, c := range parent.Children {
		if c == target {
			continue
		}
		newChildren = append(newChildren, c)
	}
	parent.Children = newChildren
	// No structural validation needed: deletion can never introduce a new
	// duplicate, a new project, or a new nesting.
	return dup, nil
}

// ApplySubtreeMove returns a new tree with the subtree at srcPath relocated
// under destParentPath. Rejects self-move and moves-into-own-descendant
// with SHARE_CYCLE.
func ApplySubtreeMove(t *StructuredShareTree, srcPath, destParentPath string, opts *ShareTreeValidationOptions) (*StructuredShareTree, error) {
	if t == nil || t.Root == nil {
		return nil, ErrNoShareTree
	}

	srcNorm, err := NormalizeSharePath(srcPath)
	if err != nil {
		return nil, pathNotFound(srcPath)
	}
	destNorm, err := NormalizeSharePath(destParentPath)
	if err != nil {
		return nil, pathNotFound(destParentPath)
	}

	dup := &StructuredShareTree{Root: CloneShareTreeSubtree(t.Root)}
	src, srcParent, err := FindNodeByPath(dup.Root, srcNorm)
	if err != nil {
		return nil, pathNotFound(srcNorm)
	}
	if srcParent == nil {
		return nil, &ShareTreeValidationErrors{Errs: []ShareTreeValidationError{{
			Code:    ShareCodeRootDelete,
			Path:    srcNorm,
			Message: "cannot move the root",
		}}}
	}
	destParent, _, err := FindNodeByPath(dup.Root, destNorm)
	if err != nil {
		return nil, pathNotFound(destNorm)
	}

	// Cycle check: dest must not be src nor a descendant of src.
	if destParent == src || IsDescendant(src, destParent) {
		return nil, &ShareTreeValidationErrors{Errs: []ShareTreeValidationError{{
			Code:    ShareCodeCycle,
			Path:    srcNorm,
			Message: fmt.Sprintf("cannot move %s into itself or its own descendant %s", srcNorm, destNorm),
		}}}
	}

	// Detach from the current parent.
	newSiblings := make([]*StructuredShareTreeNode, 0, len(srcParent.Children)-1)
	for _, c := range srcParent.Children {
		if c == src {
			continue
		}
		newSiblings = append(newSiblings, c)
	}
	srcParent.Children = newSiblings
	// Attach to the new parent (same pointer; not cloned).
	destParent.Children = append(destParent.Children, src)
	if errs := ValidateShareTree(dup, opts); len(errs) > 0 {
		return nil, &ShareTreeValidationErrors{Errs: errs}
	}
	return dup, nil
}

// SubtreeOpKind selects the mutation performed by a SubtreeOp.
type SubtreeOpKind int

const (
	// SubtreeOpReplace replaces the subtree at Path with Subtree.
	SubtreeOpReplace SubtreeOpKind = iota + 1
	// SubtreeOpAdd inserts Subtree as a new child of Path (parent).
	SubtreeOpAdd
	// SubtreeOpDelete removes the subtree at Path.
	SubtreeOpDelete
	// SubtreeOpMove relocates the subtree at Path to DestParentPath.
	SubtreeOpMove
)

// SubtreeOp describes a single share-tree mutation for ApplySubtreeBatch.
// Fields irrelevant to a given Kind are ignored.
type SubtreeOp struct {
	Kind           SubtreeOpKind
	Path           string                   // source path (Replace, Delete, Move) or parent path (Add)
	DestParentPath string                   // Move only
	Subtree        *StructuredShareTreeNode // Replace, Add only
}

// ApplySubtreeBatch applies ops sequentially to t and returns the
// resulting tree. The bottleneck for a UI burst is the qconf round-
// trip pair (read tree, write tree), not in-process work, so this
// lets callers pay 1 read + 1 write for arbitrarily many edits instead
// of 2N.
//
// Each op is routed through the existing single-op helper (Replace /
// Add / Delete / Move) so validation rules are applied at every step.
// The cumulative cost of N in-memory clones and validations is
// negligible compared to one network round-trip to qmaster.
//
// Returns ErrNoShareTree when t has no root. Returns a wrapped error
// from the first failing op, preserving its validation-code envelope.
func ApplySubtreeBatch(t *StructuredShareTree, ops []SubtreeOp, opts *ShareTreeValidationOptions) (*StructuredShareTree, error) {
	if t == nil || t.Root == nil {
		return nil, ErrNoShareTree
	}
	cur := t
	for i, op := range ops {
		var err error
		switch op.Kind {
		case SubtreeOpReplace:
			cur, err = ApplySubtreeReplace(cur, op.Path, op.Subtree, opts)
		case SubtreeOpAdd:
			cur, err = ApplySubtreeAdd(cur, op.Path, op.Subtree, opts)
		case SubtreeOpDelete:
			cur, err = ApplySubtreeDelete(cur, op.Path)
		case SubtreeOpMove:
			cur, err = ApplySubtreeMove(cur, op.Path, op.DestParentPath, opts)
		default:
			return nil, fmt.Errorf("share tree batch: unknown op kind %d at index %d", op.Kind, i)
		}
		if err != nil {
			return nil, fmt.Errorf("share tree batch op[%d]: %w", i, err)
		}
	}
	return cur, nil
}
