package storage

import (
	"testing"
)

// func TestStore(t *testing.T) {
// 	store := Store{}
// 	store.Init()
// 	key1 := "Waterloo"
// 	value1 := "Ontario"

// 	dep := uint64(0)
// 	loc := store.Set(key1, value1, dep, "branch")
// 	// store.AddChild(dep, loc)

// 	key2 := "Cheriton"
// 	value2 := "CS"

// 	store.Set(key2, value2, loc, "branch")
// 	// store.AddChild(loc, loc1)

// 	loc2 := store.Set(key1, value2, loc, "branch")
// 	// store.AddChild(loc, loc2)

// 	store.Set(key2, value1, loc2, "branch")
// 	// store.AddChild(loc2, loc3)

// 	// longest := store.GetLongestLeafLocation(loc)

// 	store.Set("does it", "work", loc, "longest-branch")
// 	// store.AddChild(longest, loc4)

// 	store.Set("does", "loc become 4?", loc2, "branch")

// 	// ret, err := store.Get("Waterloo", "latest")
// 	// if err == nil {
// 	// 	store.PrintNodes(ret)
// 	// }

// 	// fmt.Println(loc)
// 	store.Print()

// }

func TestTreeBuild(t *testing.T) {
	store := Store{}
	store.Init()

	key := "Country"
	value := "Canada"

	canada := store.Set(key, value, 0, "longest-branch")

	store.Set("Province", "Ontario", canada, "longest-branch")
	store.Set("Province", "British Columbia", canada, "longest-branch")

	store.Print()
}
