package storage

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/DCsunset/openwhisk-grpc/db"
	"github.com/DCsunset/openwhisk-grpc/utils"
	"github.com/willf/bloom"
)

// ChildInfo contains metadata corresponding to each child
// of a node in the tree. Currently, the metadata is the length
// of the longest branch starting at that child in that subtree
// TODO: Add a bloom filter parameter as additional metadata
type ChildInfo struct {
	subTreeLength uint64
	filter        *bloom.BloomFilter
}

type Node struct {
	Location uint64 // The location of the key
	Dep      uint64
	Children map[uint64]*ChildInfo
	Key      string
	Value    string
}

// NodeLocation objects are values in the MemLocation map
// They specify the key and offset in the Nodes struct where
// a specific commit point in the tree can be found.
type NodeLocation struct {
	key    string
	offset int
}

type Store struct {
	// Nodes is a dictionary that stores all nodes in the system.
	// It maps each key to all the nodes in the tree that correspond to
	// to it.
	Nodes map[string][]Node // all nodes
	// Map hash locations (commit points) to memory locations in the Nodes map.
	// key stores information about which key this commit point belongs to and
	// the offset specifies the index in the array that key maps to, of where
	// the node can be found.
	MemLocation map[uint64]NodeLocation
	lock        sync.RWMutex
	Size        int // Size of valid nodes
}

// Init initializes the storage system. It creates a root node
// for the storage tree at location 0 with key "". Its dependency
// is MaxUint64
func (s *Store) Init() {
	if len(s.Nodes) == 0 {
		// Create a root and map first
		s.MemLocation = make(map[uint64]NodeLocation)
		s.Nodes = make(map[string][]Node)
		root := Node{
			Dep:      math.MaxUint64,
			Location: 0,
			Children: make(map[uint64]*ChildInfo),
			Key:      "",
		}
		s.Nodes[""] = append(s.Nodes[""], root)
		s.MemLocation[0] = NodeLocation{
			key:    "",
			offset: 0,
		}
	}
}

func (s *Store) newNode(location uint64, dep uint64, key string, value string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.Size++
	node := Node{
		Location: location,
		Dep:      dep,
		Key:      key,
		Children: make(map[uint64]*ChildInfo),
		Value:    value,
	}

	// Add the new node to the array corresponding to
	// the relevant key in the Nodes map
	s.Nodes[key] = append(s.Nodes[key], node)
	memLoc := len(s.Nodes[key]) - 1

	s.MemLocation[location] = NodeLocation{
		key:    key,
		offset: memLoc,
	}
	fmt.Printf("%v", s.MemLocation[location])
}

// find a key present in a branch by traversing from a commit point
// towards the root
func (s *Store) GetFromBranch(key string, loc uint64) (string, error) {
	// FIXME: Similuate disk
	time.Sleep(time.Millisecond * 10)

	var node *Node
	node = s.GetNode(loc)

	// Find till root
	for {
		if node.Key == key {
			return node.Value, nil
		}
		if node.Dep == math.MaxUint64 {
			break
		}
		node = s.GetNode(node.Dep)
	}
	return "", fmt.Errorf("Key %s not found", key)
}

func (s *Store) Get(key string, heuristic string) ([]Node, error) {
	// If the key exists in the map, return nodes based on
	// heuristic
	if _, ok := s.Nodes[key]; ok {
		if heuristic == "all" {
			return s.Nodes[key], nil
		} else if heuristic == "latest" {
			return []Node{s.Nodes[key][len(s.Nodes[key])-1]}, nil
		}
	}
	// if key is not in map, return error
	return []Node{{Key: ""}}, fmt.Errorf("Key %s not found", key)
}

// type Data struct {
// 	Key   string
// 	Value string
// 	Dep   int64
// }

func (self *Store) AddChild(location uint64, child uint64) *Node {
	// get parent
	node := self.GetNode(location)
	childNode := self.GetNode(child)
	key := childNode.Key

	// update children
	_, childBranchLength := self.getLongestChild(childNode)
	n := uint(100) // No of bits per entry
	// 20 entries expected in the filter
	// 5 hash functions
	node.Children[child] = &ChildInfo{subTreeLength: childBranchLength + 1, filter: bloom.New(20*n, 5)}
	node.Children[child].filter.Add([]byte(key))
	// node.Children[child].subTreeLength = childBranchLength + 1
	// node.Children = append(node.Children, child)

	// update the length of this branch
	// for all nodes till we reach the root
	currNode := node

	for {
		if currNode.Dep == math.MaxUint64 {
			break
		}
		parent := self.GetNode(currNode.Dep)
		_, maxChildofCurrNode := self.getLongestChild(currNode)
		if parent.Children[currNode.Location].subTreeLength >= (maxChildofCurrNode + 1) {
			break
		}
		parent.Children[currNode.Location].subTreeLength = maxChildofCurrNode + 1
		// Add the key of this new child as an entry
		// in currNode's filter
		parent.Children[currNode.Location].filter.Add([]byte(key))
		currNode = parent
	}

	return node
}

// Create a new commit point in the storage system. This function creates a new
// key-value pair and adds it to the tree based on the heuristic that has been
// specified.
func (s *Store) Set(key string, value string, dep uint64, heuristic string) uint64 {
	// FIXME: Similuate disk
	time.Sleep(time.Millisecond * 10)

	if heuristic == "longest-branch" {
		dep = s.GetLongestLeafLocation(dep)
	}
	// FIX ME: Option to set the key in all possible branched reachable from dep
	// else if heuristic == "all" {
	// 	var return_list *[]uint64
	// 	return_list = &{}

	// 	// Add the node as a child of of all non-conflicting
	// 	// branches starting from dep

	// }

	return s.set(key, value, dep)
}

func (s *Store) set(key string, value string, dep uint64) uint64 {
	// Use random number + key hash
	loc := uint64(rand.Uint32()) + (uint64(utils.Hash2Uint(utils.Hash([]byte(key)))) << 32)
	// Create Node in the server at the
	// specified location
	s.newNode(loc, dep, key, value)

	// Add the newly created node as a child
	// of its computed parent
	s.AddChild(dep, loc)

	return loc
}

func (s *Store) updateNodeFilter(node *Node, value string) {
	if len(node.Children) == 0 {

	}

	for childCommit, metadata := range node.Children {
		if metadata.filter.Test([]byte(value)) {
			s.updateNodeFilter(s.GetNode(childCommit), value)
		}
	}

}

// Get the parent of a node by specifying the node's
// commit point
func (s *Store) GetParent(nodeLocation uint64) *Node {
	node := s.GetNode(nodeLocation)

	return s.GetNode(node.Dep)
}

func CreateNode(key, value string, dep uint64) *db.Node {
	// Use random number + key hash
	loc := uint64(rand.Uint32()) + (uint64(utils.Hash2Uint(utils.Hash([]byte(key)))) << 32)
	return &db.Node{
		Location: loc,
		Dep:      dep,
		Key:      key,
		Value:    value,
		Children: nil,
	}
}

// GetLongestLeafLocation returns the location of the leaf
// of the longest branch from the commit point 'dependency'
// passed to the function.
func (s *Store) GetLongestLeafLocation(dependency uint64) uint64 {
	currNode := s.GetNode(dependency)

	for {
		if len(currNode.Children) == 0 {
			break
		}
		nextNode, _ := s.getLongestChild(currNode)
		currNode = s.GetNode(nextNode)
	}
	return currNode.Location
}

func (s *Store) getLongestChild(node *Node) (maxChild uint64, maxLength uint64) {
	maxLength = 0
	maxChild = 0
	for child, metadata := range node.Children {
		if metadata.subTreeLength >= maxLength {
			maxLength = metadata.subTreeLength
			maxChild = child
		}
	}
	return maxChild, maxLength

}

// Get a node from the storage system by
// specifying its commit point
func (s *Store) GetNode(loc uint64) *Node {
	nodeLocation, ok := s.MemLocation[loc]
	if !ok {
		return nil
	}
	return &s.Nodes[nodeLocation.key][nodeLocation.offset]
}

// Add a node to the storage system
func (s *Store) AddNode(node *db.Node) {
	s.newNode(node.Location, node.Dep, node.Key, node.Value)
}

// Remove a node from the system. Currently does not work
// It does not free() the memory location but just sets
// the nodes metadata to ""
func (s *Store) RemoveNode(location uint64) {
	loc := s.MemLocation[location]
	// Seems kinda random
	// fix later!
	s.Nodes[loc.key][loc.offset] = Node{
		Key: "",
	}
}

// Print all nodes in the storage system
func (s *Store) Print() {
	fmt.Println("Nodes:")
	for key, nodes := range s.Nodes {
		fmt.Printf("Key: %s\n", key)
		for _, node := range nodes {
			if len(node.Key) > 0 {
				fmt.Printf("%s (Location: %d, Dep: %d,Value: %s,Children: {", node.Key, node.Location, node.Dep, node.Value)
				for child, metadata := range node.Children {
					fmt.Printf("%d : %d ", child, metadata.subTreeLength)
				}
				fmt.Printf("})\n")
			}
		}

	}

}

// Print the nodes from a given array
func (s *Store) PrintNodes(nodes []Node) {
	fmt.Println("Nodes:")
	for _, node := range nodes {
		if len(node.Key) > 0 {
			fmt.Printf("%s (Location: %d, Dep: %d,Value: %s,Children: {", node.Key, node.Location, node.Dep, node.Value)
			for child, metadata := range node.Children {
				fmt.Printf("%d : length : %d ", child, metadata.subTreeLength)
			}
			fmt.Printf("})\n")
		}
	}
}
