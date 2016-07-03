/*
Provides a map that can be used in a protected and concurrent fashion.
Map key must be a string, but the data can be anything.

Let's start by first getting a new map and adding a few new things to it.
You must get a map by using the New() function, and can add things to it using
Set(Key, Value):

	pm := protectedmap.New()
	pm.Set("one", 1)
	pm.Set("two", 2)

Now we can access our data through the Get function:

	data := pm.Get("one")

*/
package protectedmap

import (
	"sync"
)

// A map structure that stores data in a protected manner
type ProtectedMap struct {
	data map[string]interface{}
	lock sync.RWMutex
}

// Create a new protected map object
func New() ProtectedMap {
	return ProtectedMap{
		data: make(map[string]interface{}),
	}
}

// Add an object onto the map
func (m *ProtectedMap) Set(key string, value interface{}) {
	m.lock.Lock()
	m.data[key] = value
	m.lock.Unlock()
}

// Get a specific object out of the map.  In the event the key does not exist or
// the data is out of range, the function will have a second return of false.
//
// Get data:
// 	data, ok := pm.Get("mykey")
//
// Test if key exists:
// 	if _, ok := om.Get("mykey"); ok {
// 		... DO SOMETHING HERE ...
// 	}
func (m ProtectedMap) Get(key string) (interface{}, bool) {
	m.lock.RLock()
	data, ok := m.data[key]
	m.lock.RUnlock()
	return data, ok
}

// Delete a specific key and all associated data from the map
func (m *ProtectedMap) Delete(key string) {
	m.lock.Lock()
	delete(m.data, key)
	m.lock.Unlock()
}

// Get the total size of the map
func (m ProtectedMap) Count() int {
	m.lock.RLock()
	cnt := len(m.data)
	m.lock.RUnlock()
	return cnt
}

// A struct used to provide the ability to loop through all items in the map
type ProtectedMapIterator struct {
	returnchan chan Tuple
	breakchan  chan bool
	data       *ProtectedMap
}

// A data structure to hold returned information on each iteration
type Tuple struct {
	Key string
	Val interface{}
}

// Returns an ProtectedMapIterator type that can be used to loop through the
// entire map.
//
// This function will return a struct with two functions that should be used
// to iterate through the map: Loop() and Break().  Loop() should be provided
// to range and will return a Tuple for each item in the map.
//
// IMPORTANT NOTE: You must use the Break() function before you use the break
// go command, otherwise you might have deadlock, race, or garbage issues.
func (m *ProtectedMap) Iterator() ProtectedMapIterator {
	return ProtectedMapIterator{
		returnchan: make(chan Tuple),
		breakchan:  make(chan bool),
		data:       m,
	}
}

// Provides access to a channel that will allow looping through the entire
// map.  Returns a channel that can be passed to range and returns a
// Tuple struct with the key and value of each item.
//
// 	iter = mymap.Iterator()
// 	for data := range iter.Loop() {
// 		fmt.Printf("%s > %v\n", data.Key, data.Val)
// 	}
func (it *ProtectedMapIterator) Loop() <-chan Tuple {
	go func() {
		for k, v := range it.data.data {
			select {
			case it.returnchan <- Tuple{k, v}:
			case <-it.breakchan:
				close(it.returnchan)
				return
			}
		}

		close(it.returnchan)
		close(it.breakchan)
	}()

	return it.returnchan
}

// Signals the iterator that you no longer want to loop, allowing us to clean
// up, stop looping, and allows the garbage collector to clean up.  Finally,
// also makes sure all channels are closed and all mutex locks are clean, so
// that there are no issues with deadlocks.
func (it *ProtectedMapIterator) Break() {
	select {
	case _, _ = <-it.breakchan:
	default:
		it.breakchan <- true
	}
}
