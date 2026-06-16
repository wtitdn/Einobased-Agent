package tools

import (
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

type ToolList struct {
	Tools []tool.BaseTool
}

func RegisterAllTools(tools *ToolList) (adk.ToolsConfig, error) {
	if tools == nil {
		return adk.ToolsConfig{}, nil
	}
	return adk.ToolsConfig{
		ToolsNodeConfig: compose.ToolsNodeConfig{
			Tools: tools.Tools,
		},
	}, nil
}
