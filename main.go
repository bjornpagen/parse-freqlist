package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"unicode"
	"unsafe"
)

func yoloString(b []byte) string {
	return *((*string)(unsafe.Pointer(&b)))
}

func open(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	buf := make([]byte, info.Size())

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	_, err = io.ReadFull(f, buf)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return buf, nil
}

// freqlist in following format:
/*
you 28787591
i 27086011
the 22761659
to 17099834
a 14484562
's 14291013
it 13631703
and 10572938
that 10203742
't 9628970
*/

type freq struct {
	Word  string
	Count uint32
}
type FreqMap map[string]uint32

func parse(raw []byte) (FreqMap, error) {
	m := make(FreqMap)

	var freqs []freq
	var total, max uint32
	for {
		idx := bytes.IndexByte(raw, '\n')
		if idx == -1 {
			break
		}

		line := raw[:idx]
		raw = raw[idx+1:]

		f, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse line: %w", err)
		}

		total += f.Count
		if f.Count > max {
			max = f.Count
		}
		freqs = append(freqs, f)
	}

	for _, f := range freqs {
		m[f.Word] = calcWordScore(f.Count)
	}

	return m, nil
}

func calcWordScore(count uint32) (score uint32) {
	countRecip := 1 / float64(count)
	countFlipped := math.MaxUint32 * countRecip

	return uint32(countFlipped)
}

func parseLine(raw []byte) (freq, error) {
	var f freq

	idx := bytes.IndexByte(raw, ' ')
	if idx == -1 {
		return f, fmt.Errorf("no space found")
	}

	f.Word = string(raw[:idx])

	val, err := strconv.ParseUint(yoloString(raw[idx+1:]), 10, 32)
	f.Count = uint32(val)
	if err != nil {
		return f, fmt.Errorf("parse count: %w", err)
	}

	return f, nil
}

// takes an arbitrary string and returns a reading difficulty score
func (m FreqMap) Score(b []byte) (float64, error) {
	b = bytes.ToLower(b)
	s := yoloString(b)

	var rbuf []rune
	var scores []uint32
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

func calculateKurtosis(wordScores []uint32) float64 {
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

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: score <path to freqlist>")
		os.Exit(1)
	}

	freqlist, err := open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	m, err := parse(freqlist)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	score, err := m.Score(b)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(score)
}
