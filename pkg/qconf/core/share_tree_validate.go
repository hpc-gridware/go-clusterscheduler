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
	"strings"
)

// Share tree validation codes. These constants match the codes surfaced by
// REST handlers so the UI can branch on a stable identifier rather than
// message text.
const (
	// ShareCodeDuplicatePath signals that two sibling nodes share the
	// same name under a common parent.
	ShareCodeDuplicatePath = "SHARE_DUPLICATE_PATH"
	// ShareCodeProjectDuplicate signals that a project-typed node
	// appears more than once anywhere in the tree.
	ShareCodeProjectDuplicate = "SHARE_PROJECT_DUPLICATE"
	// ShareCodeUserDuplicateInProject signals that a user leaf appears
	// more than once within the same project subtree.
	ShareCodeUserDuplicateInProject = "SHARE_USER_DUPLICATE_IN_PROJECT"
	// ShareCodeUserDuplicateOutside signals that a user leaf appears
	// more than once outside any project subtree.
	ShareCodeUserDuplicateOutside = "SHARE_USER_DUPLICATE_OUTSIDE_PROJECT"
	// ShareCodeUserAsInterior signals that a known user appears as an
	// interior (non-leaf) node; qmaster enforces users as leaves.
	ShareCodeUserAsInterior = "SHARE_USER_AS_INTERIOR"
	// ShareCodeLeafUnknownName signals that a leaf or project name
	// does not resolve to a known user or project in the scheduler.
	ShareCodeLeafUnknownName = "SHARE_LEAF_UNKNOWN_NAME"
	// ShareCodeProjectNested signals that a project node is declared
	// inside another project's subtree.
	ShareCodeProjectNested = "SHARE_PROJECT_NESTED"
	// ShareCodeUserNoProjectAccess signals that a user leaf under a
	// project is absent from that project's ACL.
	ShareCodeUserNoProjectAccess = "SHARE_USER_NO_PROJECT_ACCESS"
	// ShareCodeNegativeShares signals that a node's share value is
	// below zero; shares must be non-negative.
	ShareCodeNegativeShares = "SHARE_NEGATIVE_SHARES"
	// ShareCodeDefaultReserved signals that the reserved name
	// "default" was used as a project or as an interior node.
	ShareCodeDefaultReserved = "SHARE_DEFAULT_RESERVED"
	// ShareCodeCycle signals an attempted subtree move under itself
	// or under one of its own descendants.
	ShareCodeCycle = "SHARE_CYCLE"
	// ShareCodePathNotFound signals that a path argument did not
	// resolve to any node in the current tree.
	ShareCodePathNotFound = "SHARE_PATH_NOT_FOUND"
	// ShareCodeRootDelete signals an attempt to delete or move the
	// tree root via a subtree operation (use the tree-level API).
	ShareCodeRootDelete = "SHARE_ROOT_DELETE"
	// ShareCodeEmptyName signals a node whose name is the empty
	// string; names must be non-empty.
	ShareCodeEmptyName = "SHARE_EMPTY_NAME"
	// ShareCodeNilNode signals a nil pointer in the Children slice
	// of a parent node.
	ShareCodeNilNode = "SHARE_NIL_NODE"
)

// Reserved leaf name that maps unspecified users to a single entitlement.
const ShareTreeDefaultName = "default"

// ShareTreeValidationError carries a machine-readable code plus a human
// message and the offending path so callers can render inline UI hints.
type ShareTreeValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path"`
}

// Error implements the error interface so a single validation error can
// flow through normal error plumbing when convenient.
func (e ShareTreeValidationError) Error() string {
	return fmt.Sprintf("%s at %s: %s", e.Code, e.Path, e.Message)
}

// ShareTreeValidationErrors aggregates multiple validation errors into a
// single error value so mutating methods can return the whole set.
type ShareTreeValidationErrors struct {
	Errs []ShareTreeValidationError
}

func (e *ShareTreeValidationErrors) Error() string {
	if len(e.Errs) == 0 {
		return "share tree: no validation errors"
	}
	msgs := make([]string, 0, len(e.Errs))
	for _, ve := range e.Errs {
		msgs = append(msgs, ve.Error())
	}
	return "share tree: " + strings.Join(msgs, "; ")
}

// ShareTreeValidationOptions enables checks that need external state. All
// fields are optional; nil fields disable the corresponding check. The
// pure structural checks always run.
type ShareTreeValidationOptions struct {
	// KnownUsers is the set of usernames known to the scheduler. When
	// provided, leaf node names are validated against this set (skipping
	// the reserved "default" name). nil disables the check.
	KnownUsers map[string]bool
	// KnownProjects is the set of project names known to the scheduler.
	// When provided, project-typed nodes are validated against this set.
	// nil disables the check.
	KnownProjects map[string]bool
	// ProjectACLs maps a project name to the set of users allowed to
	// submit under it. When provided, user leaves inside a project
	// subtree are validated against the project's ACL. nil disables the
	// check.
	ProjectACLs map[string]map[string]bool
}

// ValidateShareTree runs all structural share-tree constraints against a
// full tree and returns every violation found. An empty slice means the
// tree is valid.
//
// Rules enforced:
//   - No sibling nodes share a name (SHARE_DUPLICATE_PATH).
//   - Projects appear at most once tree-wide (SHARE_PROJECT_DUPLICATE).
//   - A user may appear at most once inside a given project subtree
//     (SHARE_USER_DUPLICATE_IN_PROJECT).
//   - A user outside any project subtree may appear at most once
//     (SHARE_USER_DUPLICATE_OUTSIDE_PROJECT).
//   - User-typed nodes must be leaves (SHARE_USER_AS_INTERIOR).
//   - Project subtrees must not contain nested projects
//     (SHARE_PROJECT_NESTED).
//   - Shares must be zero or positive (SHARE_NEGATIVE_SHARES).
//   - "default" is reserved: no project may be named "default", and
//     "default" may only appear as a user leaf (SHARE_DEFAULT_RESERVED).
//   - Node names must be non-empty (SHARE_EMPTY_NAME).
//   - No nil descendants (SHARE_NIL_NODE).
//
// When opts supplies KnownUsers, KnownProjects, or ProjectACLs, the
// corresponding state-dependent checks also run.
func ValidateShareTree(tree *StructuredShareTree, opts *ShareTreeValidationOptions) []ShareTreeValidationError {
	if tree == nil || tree.Root == nil {
		return nil
	}
	if opts == nil {
		opts = &ShareTreeValidationOptions{}
	}

	errs := []ShareTreeValidationError{}
	projectsSeen := map[string]string{}              // project name -> first path seen
	usersOutsideProjects := map[string]string{}      // user name -> first path seen
	usersInProject := map[string]map[string]string{} // project -> user -> path

	var walk func(n *StructuredShareTreeNode, pathSegs []string, activeProject string)
	walk = func(n *StructuredShareTreeNode, pathSegs []string, activeProject string) {
		path := "/" + strings.Join(pathSegs, "/")
		if n == nil {
			errs = append(errs, ShareTreeValidationError{
				Code: ShareCodeNilNode, Path: path,
				Message: "nil node in share tree",
			})
			return
		}

		if n.Name == "" {
			errs = append(errs, ShareTreeValidationError{
				Code: ShareCodeEmptyName, Path: path,
				Message: "node name is empty",
			})
		}
		if n.Shares < 0 {
			errs = append(errs, ShareTreeValidationError{
				Code: ShareCodeNegativeShares, Path: path,
				Message: fmt.Sprintf("shares must be non-negative, got %d", n.Shares),
			})
		}

		isLeaf := len(n.Children) == 0

		// "default" is reserved.
		if n.Name == ShareTreeDefaultName {
			if n.Type == ShareTreeNodeProject {
				errs = append(errs, ShareTreeValidationError{
					Code: ShareCodeDefaultReserved, Path: path,
					Message: "'default' is reserved and cannot be a project",
				})
			}
			if !isLeaf {
				errs = append(errs, ShareTreeValidationError{
					Code: ShareCodeDefaultReserved, Path: path,
					Message: "'default' must be a leaf node",
				})
			}
		}

		switch n.Type {
		case ShareTreeNodeProject:
			if activeProject != "" {
				errs = append(errs, ShareTreeValidationError{
					Code: ShareCodeProjectNested, Path: path,
					Message: fmt.Sprintf("project %q is nested inside project %q", n.Name, activeProject),
				})
			}
			if prev, seen := projectsSeen[n.Name]; seen {
				errs = append(errs, ShareTreeValidationError{
					Code: ShareCodeProjectDuplicate, Path: path,
					Message: fmt.Sprintf("project %q already appears at %s", n.Name, prev),
				})
			} else {
				projectsSeen[n.Name] = path
			}
			if opts.KnownProjects != nil {
				if _, ok := opts.KnownProjects[n.Name]; !ok {
					errs = append(errs, ShareTreeValidationError{
						Code: ShareCodeLeafUnknownName, Path: path,
						Message: fmt.Sprintf("project %q is not configured in the scheduler", n.Name),
					})
				}
			}
			activeProject = n.Name
		case ShareTreeNodeUser:
			// A user-typed node with children is a misuse (users are
			// leaves; interior organizational nodes are nominally also
			// type=0, but the Convention is that leaves are mapped to
			// actual users). Only flag if the name is NOT root AND the
			// name matches a known user AND there are children.
			// Conservative rule: report when there are children AND the
			// name is recognized as a user in the known-users set.
			if !isLeaf && opts.KnownUsers != nil {
				if _, ok := opts.KnownUsers[n.Name]; ok {
					errs = append(errs, ShareTreeValidationError{
						Code: ShareCodeUserAsInterior, Path: path,
						Message: fmt.Sprintf("user %q cannot be an interior node", n.Name),
					})
				}
			}
			if isLeaf && n.Name != ShareTreeDefaultName {
				// Track user-leaf uniqueness.
				if activeProject != "" {
					if _, ok := usersInProject[activeProject]; !ok {
						usersInProject[activeProject] = map[string]string{}
					}
					if prev, seen := usersInProject[activeProject][n.Name]; seen {
						errs = append(errs, ShareTreeValidationError{
							Code: ShareCodeUserDuplicateInProject, Path: path,
							Message: fmt.Sprintf("user %q already exists under project %q at %s",
								n.Name, activeProject, prev),
						})
					} else {
						usersInProject[activeProject][n.Name] = path
					}
				} else {
					if prev, seen := usersOutsideProjects[n.Name]; seen {
						errs = append(errs, ShareTreeValidationError{
							Code: ShareCodeUserDuplicateOutside, Path: path,
							Message: fmt.Sprintf("user %q already appears outside any project at %s",
								n.Name, prev),
						})
					} else {
						usersOutsideProjects[n.Name] = path
					}
				}
				// Known-users check (only applies to real user leaves,
				// not to "default" and not to organizational interior
				// nodes typed as user).
				if opts.KnownUsers != nil {
					if _, ok := opts.KnownUsers[n.Name]; !ok {
						errs = append(errs, ShareTreeValidationError{
							Code: ShareCodeLeafUnknownName, Path: path,
							Message: fmt.Sprintf("user %q is not configured in the scheduler", n.Name),
						})
					}
				}
				// Project-ACL check.
				if activeProject != "" && opts.ProjectACLs != nil {
					acl, ok := opts.ProjectACLs[activeProject]
					if ok {
						if _, allowed := acl[n.Name]; !allowed {
							errs = append(errs, ShareTreeValidationError{
								Code: ShareCodeUserNoProjectAccess, Path: path,
								Message: fmt.Sprintf("user %q has no access to project %q",
									n.Name, activeProject),
							})
						}
					}
				}
			}
		}

		// Duplicate sibling check: every child's name must be unique
		// among this node's children.
		if len(n.Children) > 0 {
			seen := map[string]struct{}{}
			for _, c := range n.Children {
				if c == nil {
					continue // handled by the recursive walk
				}
				if _, dup := seen[c.Name]; dup {
					childPath := path + "/" + c.Name
					errs = append(errs, ShareTreeValidationError{
						Code: ShareCodeDuplicatePath, Path: childPath,
						Message: fmt.Sprintf("sibling node with name %q already exists under %s",
							c.Name, path),
					})
				}
				seen[c.Name] = struct{}{}
			}
		}

		for _, c := range n.Children {
			childName := ""
			if c != nil {
				childName = c.Name
			}
			walk(c, append(pathSegs, childName), activeProject)
		}
	}

	walk(tree.Root, []string{tree.Root.Name}, "")
	return errs
}
