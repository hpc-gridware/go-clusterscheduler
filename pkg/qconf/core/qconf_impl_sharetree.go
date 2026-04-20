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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ClearUsage clears all user/project sharetree usage.
func (c *CommandLineQConf) ClearShareTreeUsage() error {
	_, err := c.RunCommand("-clearusage")
	return err
}

// ShowShareTree retrieves the entire share tree structure.
//
// Returns ErrNoShareTree (wrapped via fmt.Errorf with %w) when qmaster
// reports "no sharetree element" on stdout. Callers can use errors.Is
// to branch on the empty-tree case without string-matching.
func (c *CommandLineQConf) ShowShareTree() (string, error) {
	stree, err := c.RunCommand("-sstree")
	if err != nil {
		if strings.Contains(err.Error(), "no sharetree") {
			return "", fmt.Errorf("%w: %s", ErrNoShareTree, strings.TrimSpace(stree))
		}
		return "", err
	}
	return stree, nil
}

// stripRootPrefix removes the leading "/Root" component from a share-tree
// path because qconf -astnode / -dstnode / -mstnode / -sstnode reject
// paths that include it. "/Root/P1/default" becomes "/P1/default";
// "/Root" becomes "/"; "/P1" stays "/P1". Callers with already-stripped
// paths are unaffected.
func stripRootPrefix(p string) string {
	if p == "/Root" || p == "Root" {
		return "/"
	}
	if strings.HasPrefix(p, "/Root/") {
		return p[len("/Root"):]
	}
	if strings.HasPrefix(p, "Root/") {
		return "/" + p[len("Root/"):]
	}
	return p
}

// ModifyShareTreeNodes modifies the share of specified nodes in the share tree.
//
// qconf -mstnode performs *best-effort* updates: given a/b=1,bogus=2 it
// silently applies the valid entries, prints "Unable to locate bogus"
// to stdout and exits zero. We detect that phrase in the output and
// surface it as an error so the caller cannot miss partial failures.
func (c *CommandLineQConf) ModifyShareTreeNodes(nodeShareList []ShareTreeNode) error {
	if len(nodeShareList) == 0 {
		return fmt.Errorf("no nodes to modify")
	}
	var nodeShares []string
	for _, node := range nodeShareList {
		if err := ValidateSharePath(node.Node); err != nil {
			return err
		}
		nodeShares = append(nodeShares, fmt.Sprintf("%s=%d", stripRootPrefix(node.Node), node.Share))
	}
	out, err := c.RunCommand("-mstnode", strings.Join(nodeShares, ","))
	if err != nil {
		return err
	}
	if strings.Contains(out, "Unable to locate") {
		return fmt.Errorf("share tree node not found: %s", strings.TrimSpace(out))
	}
	return nil
}

// DeleteShareTreeNodes removes specified nodes from the share tree.
func (c *CommandLineQConf) DeleteShareTreeNodes(nodeList []string) error {
	if len(nodeList) == 0 {
		return fmt.Errorf("no nodes to delete")
	}
	args := make([]string, 0, len(nodeList)+1)
	args = append(args, "-dstnode")
	for _, p := range nodeList {
		if err := ValidateSharePath(p); err != nil {
			return err
		}
		args = append(args, stripRootPrefix(p))
	}
	_, err := c.RunCommand(args...)
	return err
}

// AddShareTreeNode adds a new node to the share tree, or upserts the
// share value when the node already exists (qconf -astnode silently
// overwrites). The node.Node may be either "/P1" or "/Root/P1"; any
// leading "/Root" is stripped before the qconf -astnode invocation
// because the CLI rejects that prefix.
func (c *CommandLineQConf) AddShareTreeNode(node ShareTreeNode) error {
	if err := ValidateSharePath(node.Node); err != nil {
		return err
	}
	_, err := c.RunCommand("-astnode", fmt.Sprintf("%s=%d", stripRootPrefix(node.Node), node.Share))
	return err
}

// ShowShareTreeNodes retrieves information about specified nodes or all
// nodes in the share tree. Input paths may include "/Root/..."; the
// prefix is stripped before the qconf -sstnode call.
func (c *CommandLineQConf) ShowShareTreeNodes(nodeList []string) ([]ShareTreeNode, error) {
	args := []string{"-sstnode"}
	if len(nodeList) == 0 {
		args = append(args, "/")
	} else {
		for _, p := range nodeList {
			if err := ValidateSharePath(p); err != nil {
				return nil, err
			}
			args = append(args, stripRootPrefix(p))
		}
	}
	output, err := c.RunCommand(args...)
	if err != nil {
		return nil, err
	}
	return parseShareTreeNodes(output), nil
}

// ModifyShareTree modifies the entire share tree configuration. If the
// shareTreeConfig is empty, the share tree is deleted.
// A share tree has typically the following format:
// id=0
// name=Root
// type=0
// shares=1
// childnodes=1,2,3
// where childnodes is a comma separated list of child nodes.
func (c *CommandLineQConf) ModifyShareTree(shareTreeConfig string) error {
	// if shareTreeConfig is empty, delete the share tree
	if shareTreeConfig == "" {
		// qconf -dstree
		_, err := c.RunCommand("-dstree")
		if err != nil {
			// Ignore "sharetree does not exist"
			if !strings.Contains(err.Error(), "sharetree does not exist") {
				return err
			}
		}
		return nil
	}

	file, err := CreateTempDirWithFileName("sharetree")
	if err != nil {
		return err
	}
	// Always close the file descriptor even on early error; the
	// defer runs after WriteString so the fd does not leak when
	// the write fails. The directory cleanup happens regardless.
	defer func() {
		_ = file.Close()
		_ = os.RemoveAll(filepath.Dir(file.Name()))
	}()

	if _, err := file.WriteString(shareTreeConfig); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	_, err = c.RunCommand("-Mstree", file.Name())
	return err
}

// DeleteShareTree deletes the share tree. When no tree is configured
// the call is a silent no-op (ErrNoShareTree from the existence probe
// is swallowed).
func (c *CommandLineQConf) DeleteShareTree() error {
	if _, err := c.ShowShareTree(); err != nil {
		if errors.Is(err, ErrNoShareTree) {
			return nil
		}
		return err
	}
	_, err := c.RunCommand("-dstree")
	return err
}

// ShowShareTreeStructured retrieves the share tree and parses it into
// the StructuredShareTree representation for typed access. Returns
// ErrNoShareTree (propagated via errors.Is from ShowShareTree) when
// no tree is configured.
func (c *CommandLineQConf) ShowShareTreeStructured() (*StructuredShareTree, error) {
	raw, err := c.ShowShareTree()
	if err != nil {
		return nil, err
	}
	return ParseShareTreeText(raw)
}

// ModifyShareTreeStructured replaces the entire share tree atomically. It
// serializes the structured input into the qconf text format and invokes
// qconf -Mstree via ModifyShareTree.
func (c *CommandLineQConf) ModifyShareTreeStructured(t *StructuredShareTree) error {
	txt, err := FormatShareTreeText(t)
	if err != nil {
		return err
	}
	return c.ModifyShareTree(txt)
}

// ShowShareTreeSubtree returns a deep copy of the subtree rooted at path.
// Returns ErrNoShareTree when no tree exists and a
// *ShareTreeValidationErrors (SHARE_PATH_NOT_FOUND) when the path does
// not resolve.
func (c *CommandLineQConf) ShowShareTreeSubtree(path string) (*StructuredShareTreeNode, error) {
	tree, err := c.ShowShareTreeStructured()
	if err != nil {
		return nil, err
	}
	normalized, nerr := NormalizeSharePath(path)
	if nerr != nil {
		return nil, pathNotFound(path)
	}
	target, _, ferr := FindNodeByPath(tree.Root, normalized)
	if ferr != nil {
		return nil, pathNotFound(normalized)
	}
	return CloneShareTreeSubtree(target), nil
}

// ModifyShareTreeSubtree replaces the subtree at path with sub. Composes
// ShowShareTreeStructured + ApplySubtreeReplace + ModifyShareTreeStructured.
func (c *CommandLineQConf) ModifyShareTreeSubtree(path string, sub *StructuredShareTreeNode) error {
	tree, err := c.ShowShareTreeStructured()
	if err != nil {
		return err
	}
	newTree, err := ApplySubtreeReplace(tree, path, sub, nil)
	if err != nil {
		return err
	}
	return c.ModifyShareTreeStructured(newTree)
}

// AddShareTreeSubtree inserts sub as a child of parentPath via
// read-mutate-write.
func (c *CommandLineQConf) AddShareTreeSubtree(parentPath string, sub *StructuredShareTreeNode) error {
	tree, err := c.ShowShareTreeStructured()
	if err != nil {
		return err
	}
	newTree, err := ApplySubtreeAdd(tree, parentPath, sub, nil)
	if err != nil {
		return err
	}
	return c.ModifyShareTreeStructured(newTree)
}

// DeleteShareTreeSubtree composes ApplySubtreeDelete with a read-
// mutate-write round-trip against qmaster, mirroring the pattern used
// by AddShareTreeSubtree / MoveShareTreeSubtree.
//
// The previous implementation delegated to qconf -dstnode, which
// expects paths without the `/Root` prefix — a subtle divergence from
// NormalizeSharePath that produced "node not found" against a real
// cluster even though FakeAdapter tests passed. Routing through
// qconf -Mstree unifies the code path and gives atomic semantics.
func (c *CommandLineQConf) DeleteShareTreeSubtree(path string) error {
	current, err := c.ShowShareTreeStructured()
	if err != nil {
		return err
	}
	updated, err := ApplySubtreeDelete(current, path)
	if err != nil {
		return err
	}
	return c.ModifyShareTreeStructured(updated)
}

// MoveShareTreeSubtree relocates a subtree atomically via full-tree
// replace.
func (c *CommandLineQConf) MoveShareTreeSubtree(srcPath, destParentPath string) error {
	tree, err := c.ShowShareTreeStructured()
	if err != nil {
		return err
	}
	newTree, err := ApplySubtreeMove(tree, srcPath, destParentPath, nil)
	if err != nil {
		return err
	}
	return c.ModifyShareTreeStructured(newTree)
}

// ApplyShareTreeBatch applies a sequence of subtree ops (replace, add,
// delete, copy, move) atomically: read the tree once, apply all ops in
// memory, validate, then write once via qconf -Mstree. The main win is
// collapsing 2N qmaster round-trips for a UI edit burst into 2.
//
// Returns a wrapped error from the first failing op. No partial state
// is written on failure — qconf sees the batch as a single -Mstree
// call or nothing.
func (c *CommandLineQConf) ApplyShareTreeBatch(ops []SubtreeOp) error {
	if len(ops) == 0 {
		return fmt.Errorf("share tree batch: no operations")
	}
	tree, err := c.ShowShareTreeStructured()
	if err != nil {
		return err
	}
	newTree, err := ApplySubtreeBatch(tree, ops, nil)
	if err != nil {
		return err
	}
	return c.ModifyShareTreeStructured(newTree)
}

// ShowShareTreeMonitoring runs sge_share_mon once and returns a snapshot
// of runtime share tree statistics.
//
// Errors:
//
//   - ErrNoShareTree: qmaster has no share tree configured (sge_share_mon
//     emits "No share tree" on rc=2).
//   - ErrShareTreeMonNotAvail: the sge_share_mon binary is not installed.
//     Also wraps unexpected runtime failures (non-zero exit without the
//     "No share tree" signal).
func (c *CommandLineQConf) ShowShareTreeMonitoring() (*ShareTreeMonitoring, error) {
	if c.config.DryRun {
		fmt.Printf("Executing: sge_share_mon -c 1 -n")
		return &ShareTreeMonitoring{
			CollectedAt: time.Now().UTC(),
			Nodes:       map[string]ShareTreeNodeStats{},
		}, nil
	}
	r, err := c.shareMonRunner()
	if err != nil {
		return nil, err
	}
	return ParseShareMonOutput(r)
}

// Helper function to parse the output of ShowShareTreeNodes,
// like:
// /=1
// /default=10
// /P2=11
// /P1=11
func parseShareTreeNodes(output string) []ShareTreeNode {
	lines := strings.Split(output, "\n")
	var nodes []ShareTreeNode
	for _, line := range lines {
		parts := strings.Split(line, "=")
		if len(parts) >= 2 {
			share, _ := strconv.Atoi(parts[1])
			nodes = append(nodes, ShareTreeNode{
				Node:  parts[0],
				Share: share,
			})
		}
	}
	return nodes
}
