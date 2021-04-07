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
		choices[0] = (choices[0] + 1)
		if choices[0] == limits[0] {
			choices[0] = 0 // wrap modulo
			return         // no charge
		} else {
			*n-- // charge for other updates
		}
	}
}

func chooseNth(n int, limits []int) []int {
	choices := make([]int, len(limits))
	for n > 0 {
		_chooseNth(&n, choices, limits)
		n-- // account for no wrap-around charge
	}
	return choices
}
