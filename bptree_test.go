package index

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBTree(t *testing.T) {
	testCases := []struct {
		name        string
		keysPerNode int
		toInsert    int
	}{
		{
			name:        "6 keys per node, 1000 inserts, selects, deletes",
			keysPerNode: 6,
			toInsert:    1000,
		},
		{
			name:        "5 keys per node, 100 inserts, selects, deletes",
			keysPerNode: 5,
			toInsert:    100,
		},
		{
			name:        "4 keys per node, 100 inserts, selects, deletes",
			keysPerNode: 4,
			toInsert:    100,
		},
		{
			name:        "3 keys per node, 100 inserts, selects, deletes",
			keysPerNode: 3,
			toInsert:    100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, bPlusTreeTestCase(t, tc.keysPerNode, tc.toInsert))
	}
}

func bPlusTreeTestCase(t *testing.T, keysPerNode, toInsert int) func(t *testing.T) {
	return func(t *testing.T) {
		bt := NewBPTree(keysPerNode)
		inserted := map[int]string{}
		for j := 0; j < 10; j++ {
			for i := 0; i < toInsert; i++ {
				k, v := rand.Intn(1000), randSeq(10)
				bt.Set(intKey(k), v)
				inserted[k] = v
			}

			for k, v := range inserted {
				rv, ok := bt.Get(intKey(k))
				assert.Truef(t, ok, "missing key: %d", k)
				if ok {
					assert.Equal(t, v, rv, "key with wrong value: %d", k)
				}
			}

			i := 0
			for k := range inserted {
				bt.Delete(intKey(k))
				_, ok := bt.Get(intKey(k))
				assert.False(t, ok)
				delete(inserted, k)
				i++
				if i > toInsert/2 {
					break
				}
			}

			for k, v := range inserted {
				rv, ok := bt.Get(intKey(k))
				assert.Truef(t, ok, "missing key: %d", k)
				if ok {
					assert.Equal(t, v, rv, "key with wrong value: %d", k)
				}
			}
		}

		for k := range inserted {
			bt.Delete(intKey(k))
			_, ok := bt.Get(intKey(k))
			assert.False(t, ok)
			delete(inserted, k)
		}

		assert.Empty(t, bt.ScanAll(), "everything was deleted but tree was not empty")
	}
}

func TestBTreeScan(t *testing.T) {
	bt := NewBPTree(5)

	for i, l := range letters {
		bt.Set(intKey(i), l)
	}

	assert.ElementsMatch(t, bt.ScanAll(), letters)
	assert.ElementsMatch(t, bt.Scan(2, 5), letters[2:6])
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
