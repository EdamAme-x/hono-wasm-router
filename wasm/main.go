package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall/js"
)

const METHOD_NAME_ALL = "ALL"

func splitPath(path string) []string {
	paths := strings.Split(path, "/")

	if paths[0] == "" {
		paths = paths[1:]
	}

	return paths
}

func splitRoutingPath(routePath string) []string {
	groups, path := extractGroupsFromPath(routePath)
	paths := splitPath(path)

	return replaceGroupMarks(paths, groups)
}

func extractGroupsFromPath(path string) ([][]string, string) {
	groups := [][]string{}
	re := regexp.MustCompile(`\{[^}]+\}`)

	newPath := re.ReplaceAllStringFunc(path, func(match string) string {
		mark := "@" + strconv.Itoa(len(groups))
		groups = append(groups, []string{mark, match})
		return mark
	})

	return groups, newPath
}

func replaceGroupMarks(paths []string, groups [][]string) []string {
	for i := len(groups) - 1; i >= 0; i-- {
		mark := groups[i][0]

		for j := len(paths) - 1; j >= 0; j-- {
			if strings.Contains(paths[j], mark) {
				paths[j] = strings.Replace(paths[j], mark, groups[i][1], 1)
				break
			}
		}
	}

	return paths
}

var patternCache = make(map[string]*Pattern)
var mu sync.Mutex

func getPattern(label string) *Pattern {
	if label == "*" {
		return &Pattern{
			key: label,
		}
	}

	match := regexp.MustCompile(`^\:([^\{\}]+)(?:\{(.+)\})?$`).FindStringSubmatch(label)
	if match != nil {
		mu.Lock()
		defer mu.Unlock()

		if _, exists := patternCache[label]; !exists {
			var pattern Pattern
			if match[2] != "" {
				pattern = Pattern{
					key:   label,
					value: match[1],
					regex: regexp.MustCompile("^" + match[2] + "$"),
				}
			} else {
				pattern = Pattern{
					key:     label,
					value:   match[1],
					matcher: true,
				}
			}

			patternCache[label] = &pattern
		}

		return patternCache[label]
	}

	return nil
}

type Pattern struct {
	key     string
	value   string
	matcher bool
	regex   *regexp.Regexp
}

type HandlerSet struct {
	HandlerIndex int
	PossibleKeys []string
	Score        int
}

type HandlerParamsSet struct {
	Params map[string]string
	HandlerSet
}

type Node struct {
	Methods []map[string]*HandlerSet

	Children map[string]*Node
	Patterns []Pattern
	Order    int
	Params   map[string]string
}

func (n *Node) Insert(method string, path string, handlerIndex int) *Node {
	n.Order++

	curNode := n
	parts := splitRoutingPath(path)

	possibleKeys := []string{}

	for _, part := range parts {
		if _, exists := curNode.Children[part]; exists {
			curNode = curNode.Children[part]
			pattern := getPattern(part)
			if pattern != nil {
				possibleKeys = append(possibleKeys, pattern.key)
			}

			continue
		}

		curNode.Children[part] = NewNode("", -1, map[string]*Node{})

		pattern := getPattern(part)
		if pattern != nil {
			curNode.Patterns = append(curNode.Patterns, *pattern)
			possibleKeys = append(possibleKeys, pattern.key)
		}

		curNode = curNode.Children[part]
	}

	if len(curNode.Methods) == 0 {
		curNode.Methods = []map[string]*HandlerSet{}
	}

	m := map[string]*HandlerSet{}

	parsedPossibleKeys := []string{}
	keysMap := make(map[string]bool)

	for _, possibleKey := range possibleKeys {
		if _, exists := keysMap[possibleKey]; !exists {
			keysMap[possibleKey] = true
			parsedPossibleKeys = append(parsedPossibleKeys, possibleKey)
		}
	}

	handlerSet := &HandlerSet{HandlerIndex: handlerIndex, PossibleKeys: parsedPossibleKeys, Score: n.Order}
	m[method] = handlerSet
	curNode.Methods = append(curNode.Methods, m)

	return curNode
}

func (n *Node) getHandlerSets(node *Node, method string, nodeParams map[string]string, params map[string]string) []*HandlerParamsSet {
	handlerSets := []*HandlerParamsSet{}

	for _, m := range node.Methods {
		_handlerSet := m[method]
		if _handlerSet == nil {
			_handlerSet = m[METHOD_NAME_ALL]
		}

		if _handlerSet != nil {
			processedSet := map[int]bool{}

			handlerSet := &HandlerParamsSet{
				HandlerSet: *_handlerSet,
				Params:     map[string]string{},
			}

			handlerSet.Params = map[string]string{}
			for _, key := range handlerSet.PossibleKeys {
				processed := processedSet[handlerSet.Score]
				if _, exists := params[key]; exists && !processed {
					handlerSet.Params[key] = params[key]
				} else {
					if _, exists := nodeParams[key]; exists {
						handlerSet.Params[key] = nodeParams[key]
					} else {
						handlerSet.Params[key] = params[key]
					}
				}

				processedSet[handlerSet.Score] = true
			}

			handlerSets = append(handlerSets, handlerSet)
		}
	}

	return handlerSets
}

func (n *Node) Search(method string, path string) [][]*HandlerParamsSet {
	handlerSets := []*HandlerParamsSet{}

	n.Params = map[string]string{}

	curNode := n
	curNodes := []*Node{curNode}
	parts := splitPath(path)

	for i, part := range parts {
		isLast := i == len(parts)-1
		tempNodes := []*Node{}

		for _, node := range curNodes {
			nextNode := node.Children[part]

			if nextNode != nil {
				nextNode.Params = node.Params
				if isLast {
					if _, exists := nextNode.Children["*"]; exists {
						handlerSets = append(handlerSets, n.getHandlerSets(nextNode.Children["*"], method, node.Params, map[string]string{})...)
					}
					handlerSets = append(handlerSets, n.getHandlerSets(nextNode, method, node.Params, map[string]string{})...)
				} else {
					tempNodes = append(tempNodes, nextNode)
				}
			}

			for _, pattern := range node.Patterns {
				params := map[string]string{}
				for key, value := range node.Params {
					params[key] = value
				}

				if pattern.key == "*" {
					astNode := node.Children["*"]
					if astNode != nil {
						handlerSets = append(handlerSets, n.getHandlerSets(astNode, method, node.Params, map[string]string{})...)
						tempNodes = append(tempNodes, astNode)
					}
					continue
				}

				if part == "" {
					continue
				}

				child := node.Children[pattern.key]

				restPathString := strings.Join(parts[i+1:], "/")
				if pattern.regex != nil {
					if pattern.regex.MatchString(restPathString) {
						params[pattern.value] = restPathString
						handlerSets = append(handlerSets, n.getHandlerSets(child, method, node.Params, params)...)
						continue
					}
				}

				if pattern.matcher || pattern.regex.MatchString(part) {
					params[pattern.value] = part
					if isLast {
						handlerSets = append(handlerSets, n.getHandlerSets(child, method, params, node.Params)...)
						if _, exists := child.Children["*"]; exists {
							handlerSets = append(handlerSets, n.getHandlerSets(child.Children["*"], method, params, node.Params)...)
						}
					} else {
						child.Params = params
						tempNodes = append(tempNodes, child)
					}
				}
			}
		}
		curNodes = tempNodes
	}

	results := handlerSets
	sort.Slice(results, func(i, j int) bool {
		return results[i].HandlerSet.Score < results[j].HandlerSet.Score
	})

	return [][]*HandlerParamsSet{results}
}

func NewNode(method string, handlerIndex int, children map[string]*Node) *Node {
	node := &Node{
		Order:  0,
		Params: map[string]string{},
	}

	node.Children = children

	if method == "" && handlerIndex == -1 {
		node.Methods = []map[string]*HandlerSet{}
	} else {
		m := map[string]*HandlerSet{}
		m[method] = &HandlerSet{HandlerIndex: handlerIndex, PossibleKeys: []string{}, Score: 0}
		node.Methods = []map[string]*HandlerSet{m}
	}
	node.Patterns = []Pattern{}

	return node
}

// Noop
func main() {
	c := make(chan struct{})
	<-c
}

var node = NewNode("", -1, map[string]*Node{})

//export Add
func Add(method js.Value, path js.Value, handlerIndex js.Value) {
	fmt.Println("Add", method, path, handlerIndex)
	node = node.Insert(method.String(), path.String(), handlerIndex.Int())
}

//export Match
func Match(method js.Value, path js.Value) js.Value {
	fmt.Println("Match", method, path)
	return js.ValueOf(node.Search(method.String(), path.String()))
}
