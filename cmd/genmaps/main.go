package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

type freq struct {
	Word  string
	Count uint32
}

type freqScore struct {
	Word  string
	Score float64
}

func parse(raw []byte) ([]freqScore, error) {
	var freqs []freq
	var max uint32
	for {
		idx := bytes.IndexByte(raw, '\n')
		if idx == -1 {
			break
		}

		line := raw[:idx]
		raw = raw[idx+1:]

		f, err := parseLine(line)
		if err != nil {
			if err == ErrInvalidKey {
				continue
			}
			return nil, fmt.Errorf("parse line: %w", err)
		}

		if f.Count > max {
			max = f.Count
		}

		freqs = append(freqs, f)
	}

	var scores []freqScore
	for i := range freqs {
		score := calcWordScore(freqs[i].Count, max)
		scores = append(scores, freqScore{
			Word:  freqs[i].Word,
			Score: score,
		})
	}

	// sort them by score
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score < scores[j].Score
	})

	return scores, nil
}

func calcWordScore(count, max uint32) (score float64) {
	countRecip := 1 / float64(count) // countRecip bound to [1/float64(max), 1]
	return countRecip
}

var ErrInvalidKey = fmt.Errorf("invalid key")

func parseLine(raw []byte) (freq, error) {
	var f freq

	idx := bytes.IndexByte(raw, ' ')
	if idx == -1 {
		return f, fmt.Errorf("no space found")
	}

	f.Word = string(raw[:idx])
	if strings.ContainsAny(f.Word, " 0123456789-.") {
		return f, ErrInvalidKey
	}

	val, err := strconv.ParseUint(yoloString(raw[idx+1:]), 10, 32)
	f.Count = uint32(val)
	if err != nil {
		return f, fmt.Errorf("parse count: %w", err)
	}

	return f, nil
}

func genMap(input, lang string) error {
	raw, err := open(input)
	if err != nil {
		return fmt.Errorf("open %s: %w", input, err)
	}

	outputDir := filepath.Join("..", "..", "lang", lang)

	wordMap, err := parse(raw)
	if err != nil {
		return fmt.Errorf("parse %s: %w", input, err)
	}

	// Generate the go source file containing the autogenMap
	source := freqsToSource(wordMap, lang)

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", outputDir, err)
	}

	outputFile := filepath.Join(outputDir, "autogen.go")

	err = ioutil.WriteFile(outputFile, []byte(source), 0644)
	if err != nil {
		return fmt.Errorf("write %s: %w", outputFile, err)
	}

	return nil
}

func freqsToSource(scores []freqScore, lang string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "package %s\n\n", lang)

	fmt.Fprintf(&b, "import \"github.com/bjornpagen/parse-freqlist/freqmap\"\n\n")

	fmt.Fprintf(&b, "var Map = %s\n", freqsToString(scores))

	return b.String()
}

func freqsToString(scores []freqScore) string {
	var b strings.Builder

	fmt.Fprintf(&b, "freqmap.FreqMap{\n")
	for _, f := range scores {
		fmt.Fprintf(&b, "\t%q: %x,\n", f.Word, f.Score)
	}
	fmt.Fprintf(&b, "}")

	return b.String()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: score <path to freqlist>")
		os.Exit(1)
	}

	path := os.Args[1]
	// /Users/bjornpagen/Code/FrequencyWords/content/2018/en/en_full.txt
	lastSlash := strings.LastIndexByte(path, '/')
	if lastSlash == -1 {
		fmt.Println("invalid path")
		os.Exit(1)
	}

	locale := path[lastSlash-2 : lastSlash]

	err := genMap(os.Args[1], locale)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
