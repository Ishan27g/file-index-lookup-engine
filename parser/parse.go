package parser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	
	tree "github.com/emirpasic/gods/trees/avltree"
)
var chars = []string{"a","b","c","d","e","f","g","h","i","j","k","l","m","n",
	"o","p","q","r","s","t","u","v","w","x","y","z","0","1","2","3","4","5",
	"6","7","8","9"}


type SFile struct {
	Filename  string
	FileLoc   string
	LineNum   []int32
	WordIndex []int32
}
type Parser struct {
	source string
	lock sync.Mutex
	tree *tree.Tree
}
func Start(inst string)*Parser {
	avl1 := tree.NewWithIntComparator()
	for _, char := range chars {
		avl2 := tree.NewWithStringComparator()
		avl1.Put(int(char[0]), avl2)
	}
	return &Parser{
		source: inst,
		lock: sync.Mutex{},
		tree: avl1,
	}
}
func (p *Parser)AddFile(filePath string)bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			return
		}
	}(file)
	
	lineNum := 0
	lines := make(map[int][]string)
	words := make(map[string]*SFile)
	
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lineWords := strings.Split(scanner.Text(), " ")
		if len(lineWords) == 0 {
			continue
		}
		for _, word := range lineWords {
			words[word] = &SFile{
				Filename:  filePath,
				FileLoc:   p.source,
				LineNum:   nil,
				WordIndex: nil,
			}
		}
		lines[lineNum] = lineWords
		lineNum++
	}
	words = buildWordMap(lines, words)
	p.lock.Lock()
	buildTree(p.tree, words)
	p.lock.Unlock()
	return true
}

func buildWordMap(lines map[int][]string, words map[string]*SFile) map[string]*SFile {
	for lineNum, lineStr := range lines {
		for i, word := range lineStr {
			wordLoc := words[word]
			wordLoc.WordIndex = append(wordLoc.WordIndex, int32(i))
			wordLoc.LineNum = append(wordLoc.LineNum, int32(lineNum))
			words[word] = wordLoc
		}
	}
	return words
}

func buildTree(t *tree.Tree, words map[string]*SFile) {
	for word, sf := range words {
		w := int(word[0])
		child, found := t.Get(w)
		if found {
			switch c := child.(type) {
			case *tree.Tree:
				sfList, found := c.Get(word)
				if found {
					switch sfl := sfList.(type) {
					case []*SFile:
						sfl = append(sfl, sf)
						c.Put(word, sfl)
					}
				}else {
					var sfl []*SFile
					sfl = append(sfl, sf)
					c.Put(word, sfl)
				}
				t.Put(w,c)
			}
		}
	}
}
func PrintFull(parser *Parser){
	parser.lock.Lock()
	for tre := parser.tree.Iterator(); tre.Next();{
		char := tre.Key()
		wordTree := tre.Value()
		switch wd := wordTree.(type) {
		case *tree.Tree:
			for w := wd.Iterator(); w.Next();{
				word := w.Key()
				sfList := w.Value()
				switch sfl := sfList.(type) {
				case []*SFile:
					fmt.Println("Char ", char, " word ", word)
					for sf := range sfl{
						fmt.Println("\t", sf)
					}
				}
			}
		}
	}
	parser.lock.Unlock()
}
func (p *Parser)Find(word string) *[]SFile {
	p.lock.Lock()
	child, found := p.tree.Get(int(word[0])) // char
	p.lock.Unlock()
	fmt.Println(child,found)
	var sFile []SFile
	if found {
		switch c := child.(type) {
		case *tree.Tree:
			sfList, found := c.Get(word) // word
			if found {
				switch sfl := sfList.(type) {
				case []*SFile: // list of *sf
					for _, file := range sfl {
						sFile = append(sFile, *file)
					}
				}
				fmt.Println("yayayayay")
				return &sFile
			}
		}
	}
	return nil
}
func Stringify(sfList []SFile)string{
	msg := ""
	for i, file := range sfList {
		msg += strconv.Itoa(i) + ","
		msg += file.Filename + ","
		msg += file.FileLoc + ","
		msg += "["
		for _, i3 := range file.LineNum {
			msg += strconv.Itoa(int(i3)) + ","
		}
		msg += "],["
		for _, i3 := range file.WordIndex {
			msg += strconv.Itoa(int(i3)) + ","
		}
		msg += "]\n"
	}
	return msg
}
func (sf * SFile)GetFileName()string{
	return sf.Filename
}