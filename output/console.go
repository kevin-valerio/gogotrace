package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/gogotrace/gogotrace/tree"
)

type ConsoleFormatter struct {
	writer io.Writer
}

func NewConsoleFormatter(w io.Writer) *ConsoleFormatter {
	return &ConsoleFormatter{writer: w}
}

func (cf *ConsoleFormatter) Format(callTree *tree.CallTree) error {
	if callTree.Root == nil {
		return fmt.Errorf("call tree is empty")
	}
	
	for _, child := range callTree.Root.Children {
		cf.printNode(child, "", true)
	}
	
	return nil
}

func (cf *ConsoleFormatter) printNode(node *tree.CallNode, prefix string, isLast bool) {
	indent := cf.getIndent(node.Depth - 1)
	
	line := cf.formatNodeLine(node)
	fmt.Fprintf(cf.writer, "%s%s\n", indent, line)
	
	for i, child := range node.Children {
		isLastChild := i == len(node.Children)-1
		cf.printNode(child, prefix, isLastChild)
	}
}

func (cf *ConsoleFormatter) getIndent(depth int) string {
	return strings.Repeat("    ", depth)
}

func (cf *ConsoleFormatter) formatNodeLine(node *tree.CallNode) string {
	var sb strings.Builder
	
	if node.Function.Receiver != "" {
		sb.WriteString(fmt.Sprintf("%s.%s", node.Function.Receiver, node.Function.Name))
	} else {
		sb.WriteString(node.Function.Name)
	}
	
	if node.Usages > 1 {
		sb.WriteString(fmt.Sprintf(" (%d usages)", node.Usages))
	}
	
	if strings.Contains(node.Function.Package, "github.com/") {
		sb.WriteString(fmt.Sprintf(" in %s/%s", node.Function.Package, node.Function.File))
	} else {
		sb.WriteString(fmt.Sprintf(" in %s", node.Function.FullPath))
	}
	
	return sb.String()
}