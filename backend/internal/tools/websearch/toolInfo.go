package websearch

import (
	"context"
	"einoproject/internal/model"
	"einoproject/internal/usecase"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/browseruse"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	ToolName = "browser_use"
)

func NewTool(ctx context.Context) (tool.BaseTool, error) {
	sessionTool := &SessionBrowserTool{
		browsers: make(map[string]*sessionBrowser),
	}

	template, err := sessionTool.newBrowser(ctx)
	if err != nil {
		return nil, err
	}
	defer template.Cleanup()

	info, err := template.Info(ctx)
	if err != nil {
		return nil, err
	}
	sessionTool.info = info

	return sessionTool, nil
}

type SessionBrowserTool struct {
	mu       sync.Mutex
	info     *schema.ToolInfo
	browsers map[string]*sessionBrowser
}

type sessionBrowser struct {
	tool     *browseruse.Tool
	lastUsed time.Time
}

type fallbackSearch struct {
	primary duckduckgo.Search
}

func (s fallbackSearch) TextSearch(ctx context.Context, req *duckduckgo.TextSearchRequest) (*duckduckgo.TextSearchResponse, error) {
	if s.primary != nil {
		result, err := s.primary.TextSearch(ctx, req)
		if err == nil && result != nil && len(result.Results) > 0 {
			return result, nil
		}
	}

	query := ""
	if req != nil {
		query = req.Query
	}
	if query == "" {
		return &duckduckgo.TextSearchResponse{Message: "search query is empty"}, nil
	}

	searchURL := "https://www.bing.com/search?q=" + url.QueryEscape(query)
	return &duckduckgo.TextSearchResponse{
		Message: "DuckDuckGo did not return results in time; opened a fallback search results page.",
		Results: []*duckduckgo.TextSearchResult{
			{
				Title:   "Fallback search results",
				URL:     searchURL,
				Summary: "Fallback search page for: " + query,
			},
		},
	}, nil
}

func (t *SessionBrowserTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	if t.info != nil {
		return t.info, nil
	}

	browser, err := t.newBrowser(ctx)
	if err != nil {
		return nil, err
	}
	defer browser.Cleanup()

	info, err := browser.Info(ctx)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	t.info = info
	t.mu.Unlock()
	return info, nil
}

func (t *SessionBrowserTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	sessionID := usecase.SessionIDFromContext(ctx)
	if sessionID == "" {
		return "", fmt.Errorf("browser session id missing from context")
	}

	browser, err := t.browserForSession(ctx, sessionID)
	if err != nil {
		return "", err
	}
	return browser.InvokableRun(ctx, argumentsInJSON, opts...)
}

func (t *SessionBrowserTool) browserForSession(ctx context.Context, sessionID string) (*browseruse.Tool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if item, ok := t.browsers[sessionID]; ok {
		item.lastUsed = time.Now()
		return item.tool, nil
	}

	browser, err := t.newBrowser(ctx)
	if err != nil {
		return nil, err
	}
	t.browsers[sessionID] = &sessionBrowser{
		tool:     browser,
		lastUsed: time.Now(),
	}
	return browser, nil
}

func (t *SessionBrowserTool) newBrowser(ctx context.Context) (*browseruse.Tool, error) {
	searchTool, err := duckduckgo.NewSearch(ctx, &duckduckgo.Config{
		MaxResults: 5,
		Region:     duckduckgo.RegionWT,
		Timeout:    8 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	but, err := browseruse.NewBrowserUseTool(ctx, &browseruse.Config{
		Headless:         false,
		DDGSearchTool:    fallbackSearch{primary: searchTool},
		ExtractChatModel: model.NewChatModel(),
	})
	if err != nil {
		return nil, err
	}
	return but, nil
}

func (t *SessionBrowserTool) CleanupSession(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	item, ok := t.browsers[sessionID]
	if !ok {
		return
	}
	item.tool.Cleanup()
	delete(t.browsers, sessionID)
}

func (t *SessionBrowserTool) CleanupIdle(maxIdle time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for sessionID, item := range t.browsers {
		if now.Sub(item.lastUsed) <= maxIdle {
			continue
		}
		item.tool.Cleanup()
		delete(t.browsers, sessionID)
	}
}
