package output

import (
	"fmt"
	"html/template"
	"os"

	"github.com/gogotrace/gogotrace/tree"
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoGoTrace - Call Graph</title>
    <style>
        body {
            font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
            margin: 20px;
            background-color: #f5f5f5;
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #007acc;
            padding-bottom: 10px;
        }
        .info {
            background-color: #e8f4fd;
            padding: 10px;
            border-radius: 5px;
            margin-bottom: 20px;
        }
        .tree {
            background-color: white;
            padding: 20px;
            border-radius: 5px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .node {
            margin: 5px 0;
            line-height: 1.6;
        }
        .expandable {
            cursor: pointer;
            user-select: none;
        }
        .expandable::before {
            content: '▶';
            display: inline-block;
            margin-right: 6px;
            transition: transform 0.2s;
        }
        .expandable.expanded::before {
            transform: rotate(90deg);
        }
        .children {
            margin-left: 20px;
            display: none;
        }
        .children.show {
            display: block;
        }
        .function-name {
            font-weight: bold;
            color: #007acc;
        }
        .receiver {
            color: #6a0dad;
        }
        .package {
            color: #666;
            font-size: 0.9em;
        }
        .file {
            color: #008000;
        }
        .usages {
            color: #ff6b6b;
            font-weight: bold;
        }
        .test-indicator {
            background-color: #ffd93d;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 0.8em;
            margin-left: 5px;
        }
        .controls {
            margin-bottom: 20px;
        }
        button {
            background-color: #007acc;
            color: white;
            border: none;
            padding: 8px 16px;
            margin-right: 10px;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background-color: #005a9e;
        }
        .search-box {
            padding: 8px;
            width: 300px;
            border: 1px solid #ddd;
            border-radius: 4px;
            margin-right: 10px;
        }
        .highlight {
            background-color: #ffeb3b;
            padding: 2px;
        }
    </style>
</head>
<body>
    <h1>GoGoTrace - Reverse Call Graph</h1>
    <div class="info">
        <strong>Target Function:</strong> <span class="function-name">{{.TargetSignature}}</span><br>
        <strong>Total Callers:</strong> {{.TotalCallers}}
    </div>
    <div class="controls">
        <button onclick="expandAll()">Expand All</button>
        <button onclick="collapseAll()">Collapse All</button>
        <input type="text" class="search-box" placeholder="Search functions..." onkeyup="searchFunctions(this.value)">
    </div>
    <div class="tree">
        {{.TreeHTML}}
    </div>
    <script>
        function toggleNode(element) {
            element.classList.toggle('expanded');
            const children = element.nextElementSibling;
            if (children && children.classList.contains('children')) {
                children.classList.toggle('show');
            }
        }

        function expandAll() {
            document.querySelectorAll('.expandable').forEach(node => {
                node.classList.add('expanded');
            });
            document.querySelectorAll('.children').forEach(children => {
                children.classList.add('show');
            });
        }

        function collapseAll() {
            document.querySelectorAll('.expandable').forEach(node => {
                node.classList.remove('expanded');
            });
            document.querySelectorAll('.children').forEach(children => {
                children.classList.remove('show');
            });
        }

        function searchFunctions(query) {
            const nodes = document.querySelectorAll('.node');
            nodes.forEach(node => {
                const text = node.textContent.toLowerCase();
                const hasMatch = text.includes(query.toLowerCase());
                
                if (query === '') {
                    node.innerHTML = node.innerHTML.replace(/<span class="highlight">(.*?)<\/span>/g, '$1');
                } else if (hasMatch) {
                    const regex = new RegExp('(' + query + ')', 'gi');
                    node.innerHTML = node.innerHTML.replace(regex, '<span class="highlight">$1</span>');
                }
            });
        }

        document.querySelectorAll('.expandable').forEach(node => {
            node.addEventListener('click', function() {
                toggleNode(this);
            });
        });
    </script>
</body>
</html>`

type HTMLFormatter struct {
	outputFile string
}

type HTMLData struct {
	TargetSignature string
	TotalCallers    int
	TreeHTML        template.HTML
}

func NewHTMLFormatter(outputFile string) *HTMLFormatter {
	return &HTMLFormatter{outputFile: outputFile}
}

func (hf *HTMLFormatter) Format(callTree *tree.CallTree) error {
	if callTree.Root == nil {
		return nil
	}
	
	treeHTML := hf.buildTreeHTML(callTree.Root.Children, callTree)
	
	data := HTMLData{
		TargetSignature: callTree.Root.Function.Signature,
		TotalCallers:    hf.countTotalCallers(callTree.Root),
		TreeHTML:        template.HTML(treeHTML),
	}
	
	tmpl, err := template.New("callgraph").Parse(htmlTemplate)
	if err != nil {
		return err
	}
	
	file, err := os.Create(hf.outputFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return tmpl.Execute(file, data)
}

func (hf *HTMLFormatter) buildTreeHTML(nodes []*tree.CallNode, ct *tree.CallTree) string {
	if len(nodes) == 0 {
		return ""
	}
	
	html := ""
	for _, node := range nodes {
		html += hf.buildNodeHTML(node, ct)
	}
	return html
}

func (hf *HTMLFormatter) buildNodeHTML(node *tree.CallNode, ct *tree.CallTree) string {
	hasChildren := len(node.Children) > 0
	
	nodeClass := "node"
	if hasChildren {
		nodeClass += " expandable"
	}
	
	html := fmt.Sprintf(`<div class="%s">`, nodeClass)
	
	if node.Function.Receiver != "" {
		html += fmt.Sprintf(`<span class="receiver">%s.</span>`, node.Function.Receiver)
	}
	html += fmt.Sprintf(`<span class="function-name">%s</span>`, node.Function.Name)
	
	if node.Usages > 1 {
		html += fmt.Sprintf(` <span class="usages">(%d usages)</span>`, node.Usages)
	}
	
	html += fmt.Sprintf(` in <span class="package">%s</span>/<span class="file">%s</span>`, 
		node.Function.Package, node.Function.File)
	
	if node.Function.IsTest {
		html += `<span class="test-indicator">TEST</span>`
	}
	
	html += `</div>`
	
	if hasChildren {
		html += `<div class="children">`
		html += hf.buildTreeHTML(node.Children, ct)
		html += `</div>`
	}
	
	return html
}

func (hf *HTMLFormatter) countTotalCallers(root *tree.CallNode) int {
	count := 0
	visited := make(map[string]bool)
	hf.countCallersRecursive(root, &count, visited)
	return count
}

func (hf *HTMLFormatter) countCallersRecursive(node *tree.CallNode, count *int, visited map[string]bool) {
	key := fmt.Sprintf("%s.%s.%s", node.Function.Package, node.Function.Receiver, node.Function.Name)
	if visited[key] {
		return
	}
	visited[key] = true
	
	*count += len(node.Children)
	for _, child := range node.Children {
		hf.countCallersRecursive(child, count, visited)
	}
}