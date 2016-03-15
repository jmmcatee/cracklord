# ProtectedMap #
Provides a map that can be used in a protected and concurrent fashion. Map key 
must be a string, but the data can be anything.

[![GoDoc](https://godoc.org/github.com/emperorcow/protectedmap?status.svg)](http://godoc.org/github.com/emperorcow/protectedmap)
[![Build Status](https://drone.io/github.com/emperorcow/protectedmap/status.png)](https://drone.io/github.com/emperorcow/protectedmap/latest)
[![Coverage Status](https://coveralls.io/repos/emperorcow/protectedmap/badge.svg?branch=master)](https://coveralls.io/r/emperorcow/protectedmap?branch=master)

# Usage #
You'll first need to make a new map and add some data.  Create a new map using the New() function and then add some data in using the Set(Key, Val) function, which takes a key as a string, and any data type as the value:
```
pm := protectedmap.New()
pm.Set("one", TestData{ID: 1, Name: "one"})
pm.Set("two", TestData{ID: 2, Name: "two"})
pm.Set("three", TestData{ID: 3, Name: "three"})
```

You can get data from the map using Get. 

```
datakey, ok := pm.Get("two")

```

There are also many other things, you can do, like delete, get the size, etc.  See the godoc for more information

