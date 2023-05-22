package freqmap

import (
	"bytes"
	"math"
	"unicode"
	"unsafe"
)

func yoloString(b []byte) string {
	return *((*string)(unsafe.Pointer(&b)))
}

type FreqMap map[string]float64

// takes an arbitrary string and returns a reading difficulty score
func (m FreqMap) Score(b []byte) (float64, error) {
	b = bytes.ToLower(b)
	s := yoloString(b)

	var rbuf []rune
	var scores []float64
	for _, r := range s {
		if !unicode.IsLetter(r) {
			val, ok := m[string(rbuf)]
			if ok {
				scores = append(scores, val)
			}

			rbuf = rbuf[:0]
			continue
		}

		rbuf = append(rbuf, r)
	}

	return calculateKurtosis(scores), nil
}

func calculateKurtosis(wordScores []float64) float64 {
	n := len(wordScores)
	if n <= 1 {
		return 0
	}

	var sum, sumOfSquares float64
	for _, score := range wordScores {
		val := float64(score)
		sum += val
		sumOfSquares += val * val
	}

	mean := sum / float64(n)
	variance := (sumOfSquares / float64(n)) - (mean * mean)
	stdDev := math.Sqrt(variance)

	var sumOfFourthPowers float64
	for _, score := range wordScores {
		deviation := float64(score) - mean
		fourthPower := math.Pow(deviation/stdDev, 4)
		sumOfFourthPowers += fourthPower
	}

	return (1.0 / float64(n)) * sumOfFourthPowers
}
