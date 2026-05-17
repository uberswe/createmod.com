package pages

import (
	"createmod/internal/server"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

var (
	liveServerCache   []LiveServer
	liveServerCacheAt time.Time
	liveServerMu      sync.Mutex
)

const kinTilesPreviewTemplate = "./template/kin-tiles-preview.html"

var kinTilesPreviewTemplates = append([]string{
	kinTilesPreviewTemplate,
}, commonTemplates...)

type KinTilesPreviewData struct {
	DefaultData
	LiveServers []LiveServer
}

type LiveServer struct {
	Name          string `json:"name"`
	PlayersOnline int    `json:"players_online"`
	IsOnline      bool   `json:"is_online"`
}

func getLiveServers() []LiveServer {
	liveServerMu.Lock()
	defer liveServerMu.Unlock()
	if time.Since(liveServerCacheAt) < 5*time.Minute && liveServerCache != nil {
		return liveServerCache
	}
	servers := fetchLiveServersFromAPI()
	if servers != nil {
		liveServerCache = servers
		liveServerCacheAt = time.Now()
	}
	return liveServerCache
}

func fetchLiveServersFromAPI() []LiveServer {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://www.createmodservers.com/api/v1/servers?sort=votes&per_page=5")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var result struct {
		Servers []LiveServer `json:"servers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	var online []LiveServer
	for _, s := range result.Servers {
		if s.IsOnline {
			online = append(online, s)
		}
		if len(online) == 3 {
			break
		}
	}
	return online
}

func KinTilesPreviewHandler(registry *server.Registry) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := KinTilesPreviewData{}
		d.Populate(e)
		d.Title = "Kin Tiles Preview"
		d.NoIndex = true
		d.LiveServers = getLiveServers()
		html, err := registry.LoadFiles(kinTilesPreviewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
