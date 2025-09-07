package output

import (
	"encoding/json"
	"os"

	"github.com/gogotrace/gogotrace/tree"
)

type JSONNode struct {
	Name      string      `json:"name"`
	Receiver  string      `json:"receiver,omitempty"`
	Package   string      `json:"package"`
	File      string      `json:"file"`
	Line      int         `json:"line"`
	Signature string      `json:"signature"`
	Usages    int         `json:"usages,omitempty"`
	IsTest    bool        `json:"isTest,omitempty"`
	Children  []*JSONNode `json:"children,omitempty"`
}

type JSONFormatter struct {
	outputFile string
}

func NewJSONFormatter(outputFile string) *JSONFormatter {
	return &JSONFormatter{outputFile: outputFile}
}

func (jf *JSONFormatter) Format(callTree *tree.CallTree) error {
	if callTree.Root == nil {
		return nil
	}
	
	root := &JSONNode{
		Name:      callTree.Root.Function.Name,
		Receiver:  callTree.Root.Function.Receiver,
		Package:   callTree.Root.Function.Package,
		File:      callTree.Root.Function.File,
		Line:      callTree.Root.Function.Line,
		Signature: callTree.Root.Function.Signature,
		IsTest:    callTree.Root.Function.IsTest,
	}
	
	for _, child := range callTree.Root.Children {
		jsonChild := jf.buildJSONNode(child)
		root.Children = append(root.Children, jsonChild)
	}
	
	file, err := os.Create(jf.outputFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(root)
}

func (jf *JSONFormatter) buildJSONNode(node *tree.CallNode) *JSONNode {
	jsonNode := &JSONNode{
		Name:      node.Function.Name,
		Receiver:  node.Function.Receiver,
		Package:   node.Function.Package,
		File:      node.Function.File,
		Line:      node.Function.Line,
		Signature: node.Function.Signature,
		Usages:    node.Usages,
		IsTest:    node.Function.IsTest,
	}
	
	for _, child := range node.Children {
		jsonChild := jf.buildJSONNode(child)
		jsonNode.Children = append(jsonNode.Children, jsonChild)
	}
	
	return jsonNode
}