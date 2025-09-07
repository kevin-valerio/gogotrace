package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

type Function struct {
	Name       string
	Receiver   string
	Signature  string
	Package    string
	File       string
	Line       int
	IsTest     bool
	FullPath   string
}

type CallSite struct {
	Caller *Function
	Callee *Function
}

type Analyzer struct {
	functions     sync.Map // thread-safe map[string]*Function
	callGraph     sync.Map // thread-safe map[string][]*CallSite
	fileSet       *token.FileSet
	baseDir       string
	targetSig     string
	targetFound   atomic.Bool
	filesScanned  atomic.Int32
	funcsFound    atomic.Int32
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		fileSet: token.NewFileSet(),
	}
}

func (a *Analyzer) LoadPackages(dir string) error {
	a.baseDir = dir
	
	fmt.Println("Scanning for Go files...")
	
	var allFiles []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		
		if strings.Contains(path, "vendor/") || strings.Contains(path, ".git/") || 
		   strings.Contains(path, "testdata/") || strings.Contains(path, ".work/") {
			return filepath.SkipDir
		}
		
		if strings.HasSuffix(path, ".go") && 
		   !strings.HasSuffix(path, ".pb.go") && 
		   !strings.HasSuffix(path, "_gen.go") {
			allFiles = append(allFiles, path)
		}
		
		return nil
	})
	
	if err != nil {
		return err
	}
	
	fmt.Printf("Found %d Go files to analyze\n", len(allFiles))
	
	// Phase 1: Parse all function definitions in parallel
	numWorkers := runtime.NumCPU() * 2
	fmt.Printf("Phase 1: Extracting functions with %d workers...\n", numWorkers)
	
	fileChan := make(chan string, len(allFiles))
	var wg sync.WaitGroup
	
	// Start workers for function extraction
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for filePath := range fileChan {
				a.parseFileFunctionDefs(filePath)
				count := a.filesScanned.Add(1)
				if count%100 == 0 {
					fmt.Printf("  Extracted from %d/%d files...\n", count, len(allFiles))
				}
			}
		}(i)
	}
	
	// Send files to workers
	for _, file := range allFiles {
		fileChan <- file
	}
	close(fileChan)
	
	// Wait for all workers to finish
	wg.Wait()
	
	fmt.Printf("Phase 1 complete: %d functions found\n", a.funcsFound.Load())
	
	// Phase 2: Build call graph in parallel
	fmt.Printf("Phase 2: Building call graph with %d workers...\n", numWorkers)
	
	a.filesScanned.Store(0) // Reset counter
	fileChan2 := make(chan string, len(allFiles))
	
	// Start workers for call graph building
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for filePath := range fileChan2 {
				a.parseFileCallGraph(filePath)
				count := a.filesScanned.Add(1)
				if count%100 == 0 {
					fmt.Printf("  Call graph from %d/%d files...\n", count, len(allFiles))
				}
			}
		}(i)
	}
	
	// Send files to workers again
	for _, file := range allFiles {
		fileChan2 <- file
	}
	close(fileChan2)
	
	// Wait for all workers to finish
	wg.Wait()
	
	return nil
}

func (a *Analyzer) parseFileFunctionDefs(filePath string) {
	src, err := parser.ParseFile(a.fileSet, filePath, nil, 0)
	if err != nil {
		return
	}
	
	packagePath := a.getPackagePath(filePath)
	relPath, _ := filepath.Rel(a.baseDir, filePath)
	
	// Extract all function definitions
	for _, decl := range src.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			fn := a.createFunction(funcDecl, packagePath, relPath)
			if fn != nil {
				key := a.getFunctionKey(fn)
				a.functions.Store(key, fn)
				a.funcsFound.Add(1)
			}
		}
	}
}

func (a *Analyzer) parseFileCallGraph(filePath string) {
	src, err := parser.ParseFile(a.fileSet, filePath, nil, 0)
	if err != nil {
		return
	}
	
	packagePath := a.getPackagePath(filePath)
	relPath, _ := filepath.Rel(a.baseDir, filePath)
	
	// Collect local functions for this file
	var localFunctions []*Function
	for _, decl := range src.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			fn := a.createFunction(funcDecl, packagePath, relPath)
			if fn != nil {
				localFunctions = append(localFunctions, fn)
			}
		}
	}
	
	// Analyze function bodies for calls
	for _, decl := range src.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			caller := a.createFunction(funcDecl, packagePath, relPath)
			if caller != nil {
				a.analyzeFunctionBody(funcDecl, caller, localFunctions)
			}
		}
	}
}

func (a *Analyzer) getPackagePath(filePath string) string {
	relPath, _ := filepath.Rel(a.baseDir, filepath.Dir(filePath))
	
	// Try to extract proper package path
	if idx := strings.Index(filePath, "github.com/"); idx >= 0 {
		endIdx := strings.LastIndex(filePath, "/")
		if endIdx > idx {
			return filePath[idx:endIdx]
		}
	}
	
	return relPath
}

func (a *Analyzer) createFunction(fn *ast.FuncDecl, packagePath, relPath string) *Function {
	if fn == nil {
		return nil
	}
	
	pos := a.fileSet.Position(fn.Pos())
	
	f := &Function{
		Name:     fn.Name.Name,
		Package:  packagePath,
		File:     filepath.Base(relPath),
		Line:     pos.Line,
		IsTest:   a.isTestFunction(fn, relPath),
		FullPath: relPath,
	}
	
	// Extract receiver
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv := fn.Recv.List[0]
		f.Receiver = a.formatType(recv.Type)
	}
	
	// Build signature
	f.Signature = a.buildSignature(fn)
	
	return f
}

func (a *Analyzer) buildSignature(fn *ast.FuncDecl) string {
	var parts []string
	parts = append(parts, "func")
	
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv := fn.Recv.List[0]
		recvType := a.formatType(recv.Type)
		if len(recv.Names) > 0 && recv.Names[0] != nil {
			parts = append(parts, fmt.Sprintf("(%s %s)", recv.Names[0].Name, recvType))
		} else {
			parts = append(parts, fmt.Sprintf("(%s)", recvType))
		}
	}
	
	parts = append(parts, fn.Name.Name)
	
	// Parameters
	var params []string
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			paramType := a.formatType(field.Type)
			if len(field.Names) > 0 {
				for _, name := range field.Names {
					params = append(params, fmt.Sprintf("%s %s", name.Name, paramType))
				}
			} else {
				params = append(params, paramType)
			}
		}
	}
	parts = append(parts, fmt.Sprintf("(%s)", strings.Join(params, ", ")))
	
	return strings.Join(parts, " ")
}

func (a *Analyzer) analyzeFunctionBody(fn *ast.FuncDecl, caller *Function, localFuncs []*Function) {
	if fn.Body == nil {
		return
	}
	
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			a.processCallExpr(node, caller, localFuncs)
		case *ast.FuncLit:
			anonFunc := a.createAnonymousFunction(node, caller)
			if anonFunc != nil {
				a.addCallSite(caller, anonFunc)
				a.analyzeAnonFunctionBody(node, anonFunc, localFuncs)
			}
		}
		return true
	})
}

func (a *Analyzer) analyzeAnonFunctionBody(fn *ast.FuncLit, caller *Function, localFuncs []*Function) {
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			a.processCallExpr(node, caller, localFuncs)
		case *ast.FuncLit:
			anonFunc := a.createAnonymousFunction(node, caller)
			if anonFunc != nil {
				a.addCallSite(caller, anonFunc)
				a.analyzeAnonFunctionBody(node, anonFunc, localFuncs)
			}
		}
		return true
	})
}

func (a *Analyzer) processCallExpr(call *ast.CallExpr, caller *Function, localFuncs []*Function) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		// Direct function call
		targetName := fun.Name
		
		// First check local functions in same file
		for _, fn := range localFuncs {
			if fn.Name == targetName && fn.Receiver == "" {
				a.addCallSite(caller, fn)
				return
			}
		}
		
	case *ast.SelectorExpr:
		// Method call: receiver.method()
		methodName := fun.Sel.Name
		
		// Try to identify receiver type
		var receiverType string
		if ident, ok := fun.X.(*ast.Ident); ok {
			// Simple case: r.method()
			receiverType = ident.Name
		}
		
		// Check local functions for matching methods first
		found := false
		for _, fn := range localFuncs {
			if fn.Name == methodName && fn.Receiver != "" {
				// Use heuristic: if receiver variable starts with same letter as type
				if receiverType != "" && a.couldBeReceiver(receiverType, fn.Receiver) {
					a.addCallSite(caller, fn)
					found = true
				}
			}
		}
		
		// If not found locally, search globally for methods
		if !found && methodName != "" {
			a.functions.Range(func(key, value interface{}) bool {
				fn := value.(*Function)
				if fn.Name == methodName && fn.Receiver != "" {
					if receiverType == "" || a.couldBeReceiver(receiverType, fn.Receiver) {
						a.addCallSite(caller, fn)
						found = true
						return false // Stop searching after first match
					}
				}
				return true
			})
		}
		
		// Fallback: if still not found, try matching any method with the same name
		if !found && methodName != "" {
			a.functions.Range(func(key, value interface{}) bool {
				fn := value.(*Function)
				if fn.Name == methodName && fn.Receiver != "" {
					a.addCallSite(caller, fn)
					return false // Stop after first match to avoid performance issues
				}
				return true
			})
		}
	}
}

func (a *Analyzer) couldBeReceiver(varName, receiverType string) bool {
	// Improved heuristic for receiver matching
	receiverType = strings.TrimPrefix(receiverType, "*")
	
	if len(varName) > 0 && len(receiverType) > 0 {
		varLower := strings.ToLower(varName)
		typeLower := strings.ToLower(receiverType)
		
		// Exact match (case insensitive)
		if varLower == typeLower {
			return true
		}
		
		// Common Go naming pattern: variable name is often first letter(s) of type
		if strings.HasPrefix(varLower, typeLower[:1]) {
			return true
		}
		
		// Variable name contains the type (e.g., inboxMultiplexer contains inboxMultiplexer)
		if strings.Contains(varLower, typeLower) {
			return true
		}
		
		// Type contains the variable name (e.g., multiplexer in inboxMultiplexer)
		if strings.Contains(typeLower, varLower) {
			return true
		}
		
		// Check for common abbreviations
		if strings.HasSuffix(typeLower, "multiplexer") && strings.Contains(varLower, "mux") {
			return true
		}
		if strings.HasSuffix(typeLower, "multiplexer") && strings.Contains(varLower, "multiplexer") {
			return true
		}
	}
	
	return false
}

func (a *Analyzer) createAnonymousFunction(fn *ast.FuncLit, parent *Function) *Function {
	pos := a.fileSet.Position(fn.Pos())
	
	f := &Function{
		Name:     fmt.Sprintf("func(...) in %s", parent.FullPath),
		Package:  parent.Package,
		File:     parent.File,
		Line:     pos.Line,
		IsTest:   parent.IsTest,
		FullPath: parent.FullPath,
	}
	
	// Build anonymous function signature
	var paramTypes []string
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			paramType := a.formatType(field.Type)
			count := len(field.Names)
			if count == 0 {
				count = 1
			}
			for i := 0; i < count; i++ {
				paramTypes = append(paramTypes, paramType)
			}
		}
	}
	
	if len(paramTypes) > 0 {
		f.Signature = fmt.Sprintf("func(%s)", strings.Join(paramTypes, ", "))
		f.Name = fmt.Sprintf("func(%s) in %s", strings.Join(paramTypes, ", "), parent.FullPath)
	} else {
		f.Signature = "func()"
	}
	
	// Store anonymous function
	key := fmt.Sprintf("%s#anon#%d", a.getFunctionKey(parent), pos.Line)
	a.functions.Store(key, f)
	
	return f
}

func (a *Analyzer) addCallSite(caller, callee *Function) {
	if caller == nil || callee == nil {
		return
	}
	
	calleeKey := a.getFunctionKey(callee)
	
	// Get existing call sites
	var callSites []*CallSite
	if existing, ok := a.callGraph.Load(calleeKey); ok {
		callSites = existing.([]*CallSite)
	}
	
	// Check if this call site already exists
	callerKey := a.getFunctionKey(caller)
	for _, cs := range callSites {
		if a.getFunctionKey(cs.Caller) == callerKey {
			return
		}
	}
	
	// Add new call site
	callSites = append(callSites, &CallSite{
		Caller: caller,
		Callee: callee,
	})
	
	a.callGraph.Store(calleeKey, callSites)
}

func (a *Analyzer) formatType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + a.formatType(t.X)
	case *ast.SelectorExpr:
		return a.formatType(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + a.formatType(t.Elt)
		}
		return "[...]" + a.formatType(t.Elt)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", a.formatType(t.Key), a.formatType(t.Value))
	case *ast.ChanType:
		return "chan " + a.formatType(t.Value)
	case *ast.Ellipsis:
		return "..." + a.formatType(t.Elt)
	default:
		return "unknown"
	}
}

func (a *Analyzer) isTestFunction(fn *ast.FuncDecl, filename string) bool {
	if strings.HasSuffix(filename, "_test.go") {
		name := fn.Name.Name
		return strings.HasPrefix(name, "Test") ||
			strings.HasPrefix(name, "Benchmark") ||
			strings.HasPrefix(name, "Example") ||
			strings.HasPrefix(name, "Fuzz")
	}
	return false
}

func (a *Analyzer) getFunctionKey(fn *Function) string {
	if fn.Receiver != "" {
		return fmt.Sprintf("%s#%s.%s#%d", fn.Package, fn.Receiver, fn.Name, fn.Line)
	}
	return fmt.Sprintf("%s#%s#%d", fn.Package, fn.Name, fn.Line)
}

func (a *Analyzer) normalizeSignature(sig string) string {
	sig = strings.TrimSpace(sig)
	sig = strings.ReplaceAll(sig, "  ", " ")
	return sig
}

func (a *Analyzer) GetCallersOf(fn *Function) []*CallSite {
	key := a.getFunctionKey(fn)
	if val, ok := a.callGraph.Load(key); ok {
		return val.([]*CallSite)
	}
	return nil
}

func (a *Analyzer) GetFunctions() map[string]*Function {
	result := make(map[string]*Function)
	a.functions.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(*Function)
		return true
	})
	return result
}

func (a *Analyzer) GetCallGraph() map[string][]*CallSite {
	result := make(map[string][]*CallSite)
	a.callGraph.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.([]*CallSite)
		return true
	})
	return result
}

func (a *Analyzer) BuildCallGraph() error {
	// No-op as call graph is built during parsing
	return nil
}