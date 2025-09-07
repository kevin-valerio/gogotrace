package analyzer

import (
	"fmt"
	"sort"
	"strings"
)

func (a *Analyzer) FindCallers(targetSignature string, excludeTests bool) ([]*CallSite, error) {
	targetSignature = a.normalizeSignature(targetSignature)
	
	var targetFunc *Function
	var allCallSites []*CallSite
	var matchingFunctions []*Function
	
	// Search through all functions for matching signature
	a.functions.Range(func(key, value interface{}) bool {
		fn := value.(*Function)
		if a.matchesSignature(fn, targetSignature) {
			matchingFunctions = append(matchingFunctions, fn)
		}
		return true
	})
	
	// Sort matching functions for deterministic behavior
	if len(matchingFunctions) > 0 {
		// Sort by function key to ensure consistent ordering
		sort.Slice(matchingFunctions, func(i, j int) bool {
			return a.getFunctionKey(matchingFunctions[i]) < a.getFunctionKey(matchingFunctions[j])
		})
		
		// Use the first matching function as the target
		targetFunc = matchingFunctions[0]
		
		// Collect call sites from the first matching function only
		fnKey := a.getFunctionKey(targetFunc)
		if sites, ok := a.callGraph.Load(fnKey); ok {
			allCallSites = append(allCallSites, sites.([]*CallSite)...)
		}
	}
	
	if targetFunc == nil {
		return nil, fmt.Errorf("function with signature '%s' not found", targetSignature)
	}
	
	if excludeTests {
		var filtered []*CallSite
		for _, cs := range allCallSites {
			if !cs.Caller.IsTest {
				filtered = append(filtered, cs)
			}
		}
		allCallSites = filtered
	}
	
	return allCallSites, nil
}

func (a *Analyzer) matchesSignature(fn *Function, targetSignature string) bool {
	fnSig := a.normalizeSignature(fn.Signature)
	targetSig := a.normalizeSignature(targetSignature)
	
	if fnSig == targetSig {
		return true
	}
	
	fnParts := a.parseSignature(fnSig)
	targetParts := a.parseSignature(targetSig)
	
	if fnParts.name != targetParts.name {
		return false
	}
	
	if targetParts.receiver != "" {
		if !a.matchesReceiverSignature(fnParts.receiver, targetParts.receiver) {
			return false
		}
	}
	
	if targetParts.params != "" && targetParts.params != "()" {
		if !a.matchesParams(fnParts.params, targetParts.params) {
			return false
		}
	}
	
	return true
}

type signatureParts struct {
	receiver string
	name     string
	params   string
}

func (a *Analyzer) parseSignature(sig string) signatureParts {
	parts := signatureParts{}
	
	sig = strings.TrimPrefix(sig, "func")
	sig = strings.TrimSpace(sig)
	
	if strings.HasPrefix(sig, "(") {
		endRecv := strings.Index(sig, ")")
		if endRecv > 0 {
			parts.receiver = sig[1:endRecv]
			sig = strings.TrimSpace(sig[endRecv+1:])
		}
	}
	
	parenIdx := strings.Index(sig, "(")
	if parenIdx > 0 {
		parts.name = strings.TrimSpace(sig[:parenIdx])
		parts.params = sig[parenIdx:]
	} else {
		parts.name = sig
	}
	
	return parts
}

func (a *Analyzer) matchesReceiverSignature(fnReceiver, targetReceiver string) bool {
	fnReceiver = strings.TrimSpace(fnReceiver)
	targetReceiver = strings.TrimSpace(targetReceiver)
	
	fnParts := strings.Fields(fnReceiver)
	targetParts := strings.Fields(targetReceiver)
	
	if len(fnParts) < 2 || len(targetParts) < 2 {
		return false
	}
	
	fnType := fnParts[len(fnParts)-1]
	targetType := targetParts[len(targetParts)-1]
	
	fnType = strings.TrimPrefix(fnType, "*")
	targetType = strings.TrimPrefix(targetType, "*")
	
	return fnType == targetType
}

func (a *Analyzer) matchesParams(fnParams, targetParams string) bool {
	fnParams = strings.Trim(fnParams, "()")
	targetParams = strings.Trim(targetParams, "()")
	
	if fnParams == targetParams {
		return true
	}
	
	fnParamList := a.splitParams(fnParams)
	targetParamList := a.splitParams(targetParams)
	
	if len(fnParamList) != len(targetParamList) {
		return false
	}
	
	for i := range fnParamList {
		if !a.matchesParam(fnParamList[i], targetParamList[i]) {
			return false
		}
	}
	
	return true
}

func (a *Analyzer) splitParams(params string) []string {
	if params == "" {
		return []string{}
	}
	
	var result []string
	var current strings.Builder
	depth := 0
	
	for _, ch := range params {
		switch ch {
		case '(', '[', '{':
			depth++
			current.WriteRune(ch)
		case ')', ']', '}':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}
	
	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}
	
	return result
}

func (a *Analyzer) matchesParam(fnParam, targetParam string) bool {
	fnParts := strings.Fields(fnParam)
	targetParts := strings.Fields(targetParam)
	
	if len(fnParts) == 0 || len(targetParts) == 0 {
		return false
	}
	
	fnType := fnParts[len(fnParts)-1]
	targetType := targetParts[len(targetParts)-1]
	
	return fnType == targetType
}

