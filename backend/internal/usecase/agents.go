package usecase

import "sort"

type AgentListItem struct {
	Name  string
	Label string
	Path  string
}

type AgentService struct {
	agentNames []string
}

func NewAgentService(agentNames []string) *AgentService {
	names := append([]string(nil), agentNames...)
	sort.Strings(names)
	return &AgentService{agentNames: names}
}

func (as *AgentService) AgentList() []AgentListItem {
	labels := map[string]string{
		"simple":   "SSEAgent",
		"toolCall": "Tool Call",
	}

	items := make([]AgentListItem, 0, len(as.agentNames))
	for _, name := range as.agentNames {
		label := labels[name]
		if label == "" {
			label = name
		}
		items = append(items, AgentListItem{
			Name:  name,
			Label: label,
			Path:  "/sse/" + name,
		})
	}
	return items
}
