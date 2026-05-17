package main

import "math"

// normalize devuelve v / ||v||. Si la norma es 0, devuelve v sin tocar.
func normalize(v []float64) []float64 {
	var n float64
	for _, x := range v {
		n += x * x
	}
	n = math.Sqrt(n)
	if n == 0 {
		return v
	}
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = x / n
	}
	return out
}

// dot asume vectores de igual dimensión, ya normalizados (= coseno).
func dot(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	var s float64
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

// mean promedia un slice de vectores y normaliza el resultado (centroide unitario).
func mean(vs [][]float64) []float64 {
	if len(vs) == 0 {
		return nil
	}
	dim := len(vs[0])
	out := make([]float64, dim)
	for _, v := range vs {
		for i := range v {
			out[i] += v[i]
		}
	}
	for i := range out {
		out[i] /= float64(len(vs))
	}
	return normalize(out)
}
