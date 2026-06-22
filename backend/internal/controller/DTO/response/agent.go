package response

type AgentItem struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Path  string `json:"path"`
}

type AgentsResponse struct {
	Agents []AgentItem `json:"agents"`
}
