package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/gogotrace/gogotrace/tree"
)

type ConsoleFormatter struct {
	writer     io.Writer
	showParams bool
}

func NewConsoleFormatter(w io.Writer, showParams bool) *ConsoleFormatter {
	return &ConsoleFormatter{writer: w, showParams: showParams}
}

func (cf *ConsoleFormatter) Format(callTree *tree.CallTree) error {
	if callTree.Root == nil {
		return fmt.Errorf("call tree is empty")
	}
	
	for i, child := range callTree.Root.Children {
		isLast := i == len(callTree.Root.Children)-1
		cf.printNode(child, "", isLast)
	}
	
	return nil
}

func (cf *ConsoleFormatter) printNode(node *tree.CallNode, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	
	line := cf.formatNodeLine(node)
	fmt.Fprintf(cf.writer, "%s%s%s\n", prefix, connector, line)
	
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}
	
	for i, child := range node.Children {
		isLastChild := i == len(node.Children)-1
		cf.printNode(child, childPrefix, isLastChild)
	}
}

func (cf *ConsoleFormatter) formatNodeLine(node *tree.CallNode) string {
	var sb strings.Builder
	
	if node.Function.Receiver != "" {
		sb.WriteString(fmt.Sprintf("\033[1;36m%s\033[0m.\033[1;33m%s\033[0m", node.Function.Receiver, node.Function.Name))
	} else {
		sb.WriteString(fmt.Sprintf("\033[1;33m%s\033[0m", node.Function.Name))
	}
	
	if cf.showParams && node.Function.Parameters != "" {
		sb.WriteString(fmt.Sprintf("\033[35m%s\033[0m", node.Function.Parameters))
	}
	
	if node.Usages > 1 {
		sb.WriteString(fmt.Sprintf(" \033[90m(%d usages)\033[0m", node.Usages))
	}
	
	sb.WriteString(fmt.Sprintf(" \033[90m→\033[0m \033[34m%s\033[0m", node.Function.File))
	
	return sb.String()
}