package main

import (
	"math"
)

func uniqueIntSlice(strSlice []int) []int {
	keys := make(map[int]bool)
	list := []int{}
	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func mean(v []float64) float64 {
	var res float64 = 0
	var n int = len(v)
	for i := 0; i < n; i++ {
		res += v[i]
	}
	return res / float64(n)
}

func variance(v []float64) float64 {
	if len(v) <= 1 {
		return 0
	}
	var res float64 = 0
	var m = mean(v)
	var n int = len(v)
	for i := 0; i < n; i++ {
		res += (v[i] - m) * (v[i] - m)
	}
	return res / float64(n-1)
}

func std(v []float64) float64 {
	if len(v) <= 1 {
		return 0
	}
	return roundFloat(math.Sqrt(variance(v)), 2)
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func MaxInt(a, b int, num ...int) (t int) {
	t = a
	if b > t {
		t = b
	}
	for _, i := range num {
		if i > t {
			t = i
		}
	}
	return
}

func MinInt(a, b int, num ...int) (t int) {
	t = a
	if b < t {
		t = b
	}
	for _, i := range num {
		if i < t {
			t = i
		}
	}
	return
}
