package maimai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	rg "backend-go/internal/rhythmgames"
)

type Provider struct {
	base string
	ua   string
	c    *http.Client
}

func New() *Provider {
	return &Provider{
		base: "https://www.diving-fish.com",
		ua:   "RhythmGames-DXRating/1.0",
		c:    &http.Client{Timeout: 8 * time.Second},
	}
}

func (p *Provider) Key() rg.GameKey { return rg.GameKey("maimai") }

func (p *Provider) ThemePalette() *rg.Palette {
	return &rg.Palette{GradFrom: "#ff6ad5", GradTo: "#42d392"}
}

func (p *Provider) FetchRating(ctx context.Context, id rg.UserID) (*rg.RatingResult, error) {
	if id.Username == "" {
		return nil, fmt.Errorf("username required")
	}

	body, _ := json.Marshal(map[string]string{"username": id.Username})
	req, err := http.NewRequestWithContext(ctx, "POST", p.base+"/api/maimaidxprober/query/player", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.ua != "" {
		req.Header.Set("User-Agent", p.ua)
	}

	resp, err := p.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, rg.ErrUserNotFound
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("diving-fish %d: %s", resp.StatusCode, string(b))
	}

	var out struct {
		Nickname string `json:"nickname"`
		Rating   int    `json:"rating"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &rg.RatingResult{
		DisplayName: out.Nickname,
		Rating:      out.Rating,
		Meta:        map[string]any{"source": "diving-fish"},
	}, nil
}

func init() { rg.Register(New()) }
