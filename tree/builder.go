package tree

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gogotrace/gogotrace/analyzer"
)

type CallNode struct {
	Function  *analyzer.Function
	Children  []*CallNode
	Usages    int
	Depth     int
	Visited   bool
}

type CallTree struct {
	Root       *CallNode
	Analyzer   *analyzer.Analyzer
	NoTests    bool
	visitedMap map[string]bool
}

func NewCallTree(a *analyzer.Analyzer, noTests bool) *CallTree {
	return &CallTree{
		Analyzer:   a,
		NoTests:    noTests,
		visitedMap: make(map[string]bool),
	}
}

func (ct *CallTree) Build(targetSignature string) error {
	callSites, err := ct.Analyzer.FindCallers(targetSignature, ct.NoTests)
	if err != nil {
		return err
	}
	
	if len(callSites) == 0 {
		return fmt.Errorf("no callers found for signature: %s", targetSignature)
	}
	
	targetFunc := callSites[0].Callee
	ct.Root = &CallNode{
		Function: targetFunc,
		Depth:    0,
	}
	
	callerGroups := ct.groupCallSitesByCaller(callSites)
	
	for caller, sites := range callerGroups {
		childNode := &CallNode{
			Function: caller,
			Usages:   len(sites),
			Depth:    1,
		}
		ct.Root.Children = append(ct.Root.Children, childNode)
		ct.buildSubtree(childNode, 2)
	}
	
	ct.sortChildren(ct.Root)
	
	return nil
}

func (ct *CallTree) groupCallSitesByCaller(callSites []*analyzer.CallSite) map[*analyzer.Function][]*analyzer.CallSite {
	groups := make(map[*analyzer.Function][]*analyzer.CallSite)
	for _, cs := range callSites {
		groups[cs.Caller] = append(groups[cs.Caller], cs)
	}
	return groups
}

func (ct *CallTree) buildSubtree(node *CallNode, depth int) {
	if depth > 20 { // Increase depth limit for debugging
		return
	}
	
	key := ct.getFunctionKey(node.Function)
	if ct.visitedMap[key] {
		return
	}
	ct.visitedMap[key] = true
	defer func() { ct.visitedMap[key] = false }()
	
	callSites := ct.Analyzer.GetCallersOf(node.Function)
	if ct.NoTests {
		callSites = ct.filterTestCallers(callSites)
	}
	
	callerGroups := ct.groupCallSitesByCaller(callSites)
	
	for caller, sites := range callerGroups {
		childNode := &CallNode{
			Function: caller,
			Usages:   len(sites),
			Depth:    depth,
		}
		node.Children = append(node.Children, childNode)
		ct.buildSubtree(childNode, depth+1)
	}
	
	ct.sortChildren(node)
}

func (ct *CallTree) filterTestCallers(callSites []*analyzer.CallSite) []*analyzer.CallSite {
	var filtered []*analyzer.CallSite
	for _, cs := range callSites {
		if !cs.Caller.IsTest {
			filtered = append(filtered, cs)
		}
	}
	return filtered
}

func (ct *CallTree) sortChildren(node *CallNode) {
	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].Function.Package != node.Children[j].Function.Package {
			return node.Children[i].Function.Package < node.Children[j].Function.Package
		}
		if node.Children[i].Function.File != node.Children[j].Function.File {
			return node.Children[i].Function.File < node.Children[j].Function.File
		}
		if node.Children[i].Function.Name != node.Children[j].Function.Name {
			return node.Children[i].Function.Name < node.Children[j].Function.Name
		}
		// Add line number for deterministic ordering of anonymous functions
		return node.Children[i].Function.Line < node.Children[j].Function.Line
	})
}

func (ct *CallTree) getFunctionKey(fn *analyzer.Function) string {
	if fn.Receiver != "" {
		return fmt.Sprintf("%s.%s.%s", fn.Package, fn.Receiver, fn.Name)
	}
	return fmt.Sprintf("%s.%s", fn.Package, fn.Name)
}

func (ct *CallTree) FormatNode(node *CallNode) string {
	var sb strings.Builder
	
	if node.Function.Receiver != "" {
		sb.WriteString(fmt.Sprintf("%s.%s", node.Function.Receiver, node.Function.Name))
	} else {
		sb.WriteString(node.Function.Name)
	}
	
	if node.Usages > 1 {
		sb.WriteString(fmt.Sprintf(" (%d usages)", node.Usages))
	}
	
	sb.WriteString(fmt.Sprintf(" in %s", node.Function.FullPath))
	
	return sb.String()
}

func (ct *CallTree) GetDisplayName(fn *analyzer.Function) string {
	if fn.Receiver != "" {
		return fmt.Sprintf("%s.%s", fn.Receiver, fn.Name)
	}
	return fn.Name
}

func (ct *CallTree) GetFullPath(fn *analyzer.Function) string {
	return filepath.Join(fn.Package, fn.File)
}