package snapapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0xzer/snapper/data/headers"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/payload"
)

func (c *Client) GetPublicProfile(ctx context.Context, userID string) (PublicProfile, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return PublicProfile{}, fmt.Errorf("missing Snapchat user id")
	}
	if !c.needsPublicProfile(userID, true) {
		c.mu.Lock()
		profile := PublicProfile{
			UserID:    userID,
			Name:      c.namesByUserID[userID],
			AvatarURL: c.avatarsByUserID[userID],
		}
		c.mu.Unlock()
		return profile, nil
	}
	profiles, err := c.fetchPublicProfiles(ctx, []string{userID})
	if err != nil {
		return PublicProfile{}, err
	}
	profile := profiles[userID]
	return PublicProfile{
		UserID:    userID,
		Name:      profile.Name,
		AvatarURL: profile.AvatarURL,
	}, nil
}

func (c *Client) GetPublicProfiles(ctx context.Context, userIDs []string) (map[string]PublicProfile, error) {
	result := make(map[string]PublicProfile)
	toFetch := make([]string, 0, len(userIDs))
	seen := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		userID = strings.ToLower(userID)
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		if !c.needsPublicProfile(userID, true) {
			c.mu.Lock()
			result[userID] = PublicProfile{
				UserID:    userID,
				Name:      c.namesByUserID[userID],
				AvatarURL: c.avatarsByUserID[userID],
			}
			c.mu.Unlock()
			continue
		}
		toFetch = append(toFetch, userID)
	}
	if len(toFetch) == 0 {
		return result, nil
	}
	profiles, err := c.fetchPublicProfiles(ctx, toFetch)
	if err != nil {
		return nil, err
	}
	for userID, profile := range profiles {
		result[userID] = PublicProfile{
			UserID:    userID,
			Name:      profile.Name,
			AvatarURL: profile.AvatarURL,
		}
	}
	return result, nil
}

func (c *Client) CachedAvatarURL(userID string) string {
	return c.lookupAvatarURL(userID)
}

func (c *Client) DownloadAvatar(ctx context.Context, avatarURL string) ([]byte, error) {
	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL == "" {
		return nil, fmt.Errorf("missing Snapchat avatar URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, avatarURL, nil)
	if err != nil {
		return nil, err
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download Snapchat avatar returned %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("Snapchat avatar was empty")
	}
	return data, nil
}

func (c *Client) fetchPublicProfiles(ctx context.Context, userIDs []string) (map[string]publicProfile, error) {
	form, err := payload.GetPublicUserInfo(userIDs, "CHAT")
	if err != nil {
		return nil, err
	}
	header := headers.NewDefaultHeaders(c.cookies, c.tokens, true)
	c.applyUserAgentHeaders(header)
	header.Set("content-type", "application/x-www-form-urlencoded; charset=utf-8")
	body, err := c.doHTTP(ctx, paths.GET_USER_PUBLIC_INFO, http.MethodPost, header, form)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Snapchatters []json.RawMessage `json:"snapchatters"`
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	profiles := make(map[string]publicProfile, len(resp.Snapchatters))
	for _, rawUser := range resp.Snapchatters {
		var user struct {
			ID              string `json:"user_id"`
			Username        string `json:"username"`
			DisplayName     string `json:"display_name"`
			MutableUsername string `json:"mutable_username"`
			BitmojiAvatarID string `json:"bitmoji_avatar_id"`
			BitmojiSelfieID string `json:"bitmoji_selfie_id"`
		}
		if err = json.Unmarshal(rawUser, &user); err != nil {
			continue
		}
		name := user.DisplayName
		if name == "" {
			name = user.MutableUsername
		}
		if name == "" {
			name = user.Username
		}
		if user.ID != "" {
			var raw map[string]any
			_ = json.Unmarshal(rawUser, &raw)
			avatarURL := bitmojiAvatarURL(user.BitmojiSelfieID, user.BitmojiAvatarID)
			if avatarURL == "" {
				avatarURL = extractProfileAvatarURL(raw)
			}
			profiles[user.ID] = publicProfile{
				Name:      name,
				AvatarURL: avatarURL,
			}
		}
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range userIDs {
		if id = strings.TrimSpace(id); id != "" {
			c.profileCheckedAt[id] = now
		}
	}
	for id, profile := range profiles {
		if profile.Name != "" {
			c.namesByUserID[id] = profile.Name
		}
		if profile.AvatarURL != "" {
			c.avatarsByUserID[id] = profile.AvatarURL
		}
	}
	return profiles, nil
}

func (c *Client) lookupName(userID string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.namesByUserID[userID]
}

func (c *Client) lookupAvatarURL(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.avatarsByUserID[userID]
}

func (c *Client) needsPublicProfile(userID string, needsName bool) bool {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if checkedAt := c.profileCheckedAt[userID]; !checkedAt.IsZero() && time.Since(checkedAt) < minPublicProfileRefreshInterval {
		return false
	}
	if needsName && strings.TrimSpace(c.namesByUserID[userID]) == "" {
		return true
	}
	if strings.TrimSpace(c.avatarsByUserID[userID]) == "" {
		return true
	}
	return time.Since(c.profileCheckedAt[userID]) >= minPublicProfileRefreshInterval
}

func extractProfileAvatarURL(profile map[string]any) string {
	type candidate struct {
		url   string
		score int
	}
	best := candidate{}
	var walk func(any, string)
	walk = func(value any, path string) {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				nextPath := key
				if path != "" {
					nextPath = path + "." + key
				}
				walk(child, nextPath)
			}
		case []any:
			for _, child := range typed {
				walk(child, path)
			}
		case string:
			urlValue := strings.TrimSpace(typed)
			if !isHTTPURL(urlValue) {
				return
			}
			score := avatarURLScore(path, urlValue)
			if score > best.score {
				best = candidate{url: urlValue, score: score}
			}
		}
	}
	walk(profile, "")
	return best.url
}

func bitmojiAvatarURL(selfieID, avatarID string) string {
	selfieID = strings.TrimSpace(selfieID)
	avatarID = strings.TrimSpace(avatarID)
	if selfieID == "" || avatarID == "" {
		return ""
	}
	return "https://sdk.bitmoji.com/render/panel/" + url.PathEscape(selfieID+"-"+avatarID+"-v1.png") + "?transparent=1&scale=2"
}

func isHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme == "https" || parsed.Scheme == "http"
}

func avatarURLScore(path, rawURL string) int {
	lowerPath := strings.ToLower(path)
	lowerURL := strings.ToLower(rawURL)
	score := 0
	switch {
	case strings.Contains(lowerPath, "bitmoji") && strings.Contains(lowerPath, "avatar"):
		score = 100
	case strings.Contains(lowerPath, "bitmoji"):
		score = 90
	case strings.Contains(lowerPath, "avatar"):
		score = 80
	case strings.Contains(lowerPath, "profile") && (strings.Contains(lowerPath, "picture") || strings.Contains(lowerPath, "image")):
		score = 75
	case strings.Contains(lowerPath, "snapcode"):
		score = 50
	case strings.Contains(lowerPath, "thumbnail") || strings.Contains(lowerPath, "image"):
		score = 40
	}
	if score == 0 {
		if strings.Contains(lowerURL, "bitmoji") {
			score = 60
		} else if strings.Contains(lowerURL, "avatar") {
			score = 45
		}
	}
	if score > 0 && (strings.Contains(lowerPath, "url") || strings.Contains(lowerPath, "uri") || strings.Contains(lowerPath, "image")) {
		score += 5
	}
	return score
}
