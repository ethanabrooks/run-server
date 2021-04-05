package main

// IncrementArray moves one forward in an array of ints in this order:
// [0 0 0] -> [1 0 0] -> [2 0 0] -> [0 1 0] -> [0 2 0] -> ...
func _chooseNth(n *int, choices []int, limits []int) {
	if len(limits) == 0 {
		return
	}
	for {
		_chooseNth(n, choices[1:], limits[1:])
		if *n == 0 {
			return
		}
		choices[0]++
		if choices[0] == limits[0] {
			choices[0] = 0
			return
		}
		*n--
	}
}

func chooseNth(n int, limits []int) []int {
	choices := make([]int, len(limits))
	_chooseNth(&n, choices, limits)
	return choices
}
