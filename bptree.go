package index

import (
	"fmt"
)

type (
	// BPTree is all the state for a b-tree.
	BPTree struct {
		keysPerNode int
		root        bpTreeNode
	}
	bpTreeInternalNode struct {
		keys     []intKey
		children []bpTreeNode
	}
	bpTreeLeafNode struct {
		keys   []intKey
		leaves []interface{}
		prev   *bpTreeLeafNode
		next   *bpTreeLeafNode
	}
	bpTreeNode interface{}
	intKey     int
)

// NewBPTree returns a new BTree with the specified keysPerNode.
func NewBPTree(keysPerNode int) BPTree {
	return BPTree{keysPerNode: keysPerNode, root: newLeafNode(nil, nil)}
}

// Get returns the value for the given key.
func (b *BPTree) Get(key intKey) (interface{}, bool) {
	curr := b.root
	for {
		switch tCurr := curr.(type) {
		case *bpTreeInternalNode:
			curr = tCurr.children[tCurr.childIndex(key)]
		case *bpTreeLeafNode:
			for i, k := range tCurr.keys {
				if k == key {
					return tCurr.leaves[i], true
				}
			}
			return nil, false
		}
	}
}

// Set key to the given value.
func (b *BPTree) Set(key intKey, value interface{}) {
	var hist []bpTreeNode
	curr := b.root
	for {
		switch tCurr := curr.(type) {
		case *bpTreeInternalNode:
			hist = append(hist, tCurr)
			curr = tCurr.children[tCurr.childIndex(key)]
		case *bpTreeLeafNode:
			tCurr.set(key, value)
			if split := len(tCurr.keys) > b.keysPerNode; split {
				goto restructure
			}
			return
		}
	}

restructure:
	var newKey intKey
	var leftSplit, rightSplit bpTreeNode
	for {
		switch tCurr := curr.(type) {
		case *bpTreeLeafNode:
			newKey, leftSplit, rightSplit = tCurr.split()
		case *bpTreeInternalNode:
			i := tCurr.childIndex(key)
			tCurr.keys = append(tCurr.keys[:i], append([]intKey{newKey}, tCurr.keys[i:]...)...)
			tCurr.children = append(tCurr.children[:i], append([]bpTreeNode{leftSplit, rightSplit}, tCurr.children[i+1:]...)...)
			if split := len(tCurr.keys) > b.keysPerNode; !split {
				return
			}
			newKey, leftSplit, rightSplit = tCurr.split()
		}
		if len(hist) == 0 {
			b.root = newInternalNode([]intKey{newKey}, []bpTreeNode{leftSplit, rightSplit})
			return
		}
		curr = hist[len(hist)-1]
		hist = hist[:len(hist)-1]
	}
}

// Delete removes the specified key's data from the tree.
func (b *BPTree) Delete(key intKey) {
	var hist []bpTreeNode
	var internalNode *bpTreeInternalNode
	curr := b.root
	for {
		switch tCurr := curr.(type) {
		case *bpTreeInternalNode:
			hist = append(hist, tCurr)
			if _, ok := tCurr.keyIndex(key); ok {
				internalNode = tCurr
			}
			curr = tCurr.children[tCurr.childIndex(key)]

		case *bpTreeLeafNode:
			sibIndex, _ := tCurr.delete(key)
			if unsplit := len(tCurr.keys) < b.keysPerNode/2; unsplit {
				goto restructure
			}
			if internalNode != nil {
				i, _ := internalNode.keyIndex(key)
				internalNode.keys[i] = tCurr.keys[sibIndex]
			}
			return
		}
	}

restructure:
	defer func() {
		switch root := b.root.(type) {
		case *bpTreeInternalNode:
			if len(root.keys) == 0 {
				b.root = root.children[0]
			}
		}
	}()
	for {
		switch tCurr := curr.(type) {
		case *bpTreeInternalNode:
			deletedIndex := tCurr.childIndex(key)
			if deletedIndex == len(tCurr.children)-1 {
				deletedIndex--
			}
			deletedChild, deletedNeighbor := tCurr.children[deletedIndex], tCurr.children[deletedIndex+1]
			switch deletedChild := deletedChild.(type) {
			case *bpTreeInternalNode:
				newChild := deletedChild.mergeRight(tCurr.keys[deletedIndex], deletedNeighbor.(*bpTreeInternalNode))
				if len(newChild.keys) > b.keysPerNode {
					newKey, leftChild, rightChild := newChild.split()
					tCurr.keys[deletedIndex] = newKey
					tCurr.children[deletedIndex] = leftChild
					tCurr.children[deletedIndex+1] = rightChild
				} else {
					tCurr.keys = append(tCurr.keys[:deletedIndex], tCurr.keys[deletedIndex+1:]...)
					tCurr.children = append(tCurr.children[:deletedIndex], append([]bpTreeNode{newChild}, tCurr.children[deletedIndex+2:]...)...)
				}
			case *bpTreeLeafNode:
				newChild := deletedChild.mergeRight(deletedNeighbor.(*bpTreeLeafNode))
				if len(newChild.keys) > b.keysPerNode {
					newKey, leftChild, rightChild := newChild.split()
					tCurr.keys[deletedIndex] = newKey
					tCurr.children[deletedIndex] = leftChild
					tCurr.children[deletedIndex+1] = rightChild
				} else {
					tCurr.keys = append(tCurr.keys[:deletedIndex], tCurr.keys[deletedIndex+1:]...)
					tCurr.children = append(tCurr.children[:deletedIndex], append([]bpTreeNode{newChild}, tCurr.children[deletedIndex+2:]...)...)
				}
			}

			if unsplit := len(tCurr.keys) < b.keysPerNode/2; !unsplit {
				return
			}
		}
		if len(hist) == 0 {
			return
		}
		curr = hist[len(hist)-1]
		hist = hist[:len(hist)-1]
	}
}

func (n *bpTreeLeafNode) delete(key intKey) (int, bool) {
	for i, k := range n.keys {
		if key == k {
			n.keys = append(n.keys[:i], n.keys[i+1:]...)
			n.leaves = append(n.leaves[:i], n.leaves[i+1:]...)
			if i == len(n.keys) {
				return i - 1, true
			}
			return i, true
		}
	}
	return 0, false
}

// Scan returns all values in a range.
func (b *BPTree) Scan(start, end intKey) []interface{} {
	var leaf *bpTreeLeafNode
	curr := b.root
	for {
		switch tCurr := curr.(type) {
		case *bpTreeInternalNode:
			curr = tCurr.children[tCurr.childIndex(start)]
		case *bpTreeLeafNode:
			leaf = tCurr
		}
		if leaf != nil {
			break
		}
	}

	var vs []interface{}
	for {
		for i, k := range leaf.keys {
			if k > end {
				return vs
			}
			if k >= start {
				vs = append(vs, leaf.leaves[i])
			}
		}
		leaf = leaf.next
		if leaf == nil {
			break
		}
	}
	return vs
}

// ScanAll returns all the values in order.
func (b *BPTree) ScanAll() []interface{} {
	var leaf *bpTreeLeafNode
	curr := b.root
	for {
		switch tCurr := curr.(type) {
		case *bpTreeInternalNode:
			curr = tCurr.children[0]
		case *bpTreeLeafNode:
			leaf = tCurr
		}
		if leaf != nil {
			break
		}
	}

	var vs []interface{}
	for {
		vs = append(vs, leaf.leaves...)
		leaf = leaf.next
		if leaf == nil {
			break
		}
	}
	return vs
}

func (n *bpTreeLeafNode) set(key intKey, value interface{}) {
	for i, k := range n.keys {
		if key < k {
			n.keys = append(n.keys[:i], append([]intKey{key}, n.keys[i:]...)...)
			n.leaves = append(n.leaves[:i], append([]interface{}{value}, n.leaves[i:]...)...)
			return
		}
		if key == k {
			n.leaves[i] = value
			return
		}
	}
	n.keys = append(n.keys, key)
	n.leaves = append(n.leaves, value)
}

func (n *bpTreeInternalNode) keyIndex(key intKey) (int, bool) {
	for i, k := range n.keys {
		if key == k {
			return i, true
		}
	}
	return 0, false
}

func (n *bpTreeInternalNode) childIndex(key intKey) int {
	for i, k := range n.keys {
		if key < k {
			return i
		}
	}
	return len(n.keys)
}

func newLeafNode(keys []intKey, leaves []interface{}) *bpTreeLeafNode {
	nKeys := make([]intKey, len(keys))
	copy(nKeys, keys)
	nLeaves := make([]interface{}, len(leaves))
	copy(nLeaves, leaves)
	return &bpTreeLeafNode{
		keys:   nKeys,
		leaves: nLeaves,
	}
}

func newInternalNode(keys []intKey, children []bpTreeNode) *bpTreeInternalNode {
	nKeys := make([]intKey, len(keys))
	copy(nKeys, keys)
	nChildren := make([]bpTreeNode, len(children))
	copy(nChildren, children)
	return &bpTreeInternalNode{
		keys:     nKeys,
		children: nChildren,
	}
}

func (n *bpTreeInternalNode) split() (intKey, *bpTreeInternalNode, *bpTreeInternalNode) {
	split := len(n.keys) / 2
	return n.keys[split], newInternalNode(n.keys[:split], n.children[:split+1]), newInternalNode(n.keys[split+1:], n.children[split+1:])
}

func (n *bpTreeInternalNode) mergeRight(middle intKey, neighbor *bpTreeInternalNode) *bpTreeInternalNode {
	keys := append(n.keys, append([]intKey{middle}, neighbor.keys...)...)
	return newInternalNode(keys, append(n.children, neighbor.children...))
}

func (n *bpTreeLeafNode) split() (intKey, *bpTreeLeafNode, *bpTreeLeafNode) {
	split := len(n.keys) / 2
	lSplit := newLeafNode(n.keys[:split], n.leaves[:split])
	rSplit := newLeafNode(n.keys[split:], n.leaves[split:])
	if n.prev != nil {
		n.prev.next = lSplit
		lSplit.prev = n.prev
	}
	lSplit.next = rSplit
	rSplit.prev = lSplit
	if n.next != nil {
		rSplit.next = n.next
		n.next.prev = rSplit
	}
	return n.keys[split], lSplit, rSplit
}

func (n *bpTreeLeafNode) mergeRight(neighbor *bpTreeLeafNode) *bpTreeLeafNode {
	return newLeafNode(append(n.keys, neighbor.keys...), append(n.leaves, neighbor.leaves...))
}

func (b *BPTree) pprint() {
	var next []bpTreeNode
	switch curr := b.root.(type) {
	case *bpTreeInternalNode:
		fmt.Println(curr.keys)
		next = append(next, curr.children...)
	case *bpTreeLeafNode:
		fmt.Println(curr.keys)
	}

	recurse := func() []bpTreeNode {
		var nextLayer []bpTreeNode
		for _, n := range next {
			switch curr := n.(type) {
			case *bpTreeInternalNode:
				fmt.Print(curr.keys, "\t")
				nextLayer = append(nextLayer, curr.children...)
			case *bpTreeLeafNode:
				fmt.Print(curr.keys, "\t")
			}
		}
		fmt.Println()
		return nextLayer
	}

	for {
		next = recurse()
		if len(next) == 0 {
			return
		}
	}
}
