package main

// IncrementArray moves one forward in an array of ints in this order:
// [0 0 0] -> [1 0 0] -> [2 0 0] -> [0 1 0] -> [0 2 0] -> ...
// When we reached the end, this function returns true; otherwise it returns false.
func IncrementArray(values, maxValues []int) bool {
	for i := range values {
		values[i]++
		if values[i] == maxValues[i] {
			values[i] = 0
		} else {
			return false
		}
	}
	return true
}
