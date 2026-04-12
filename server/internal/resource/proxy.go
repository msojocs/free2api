package resource

import (
	"strings"
	"sync"

	"github.com/msojocs/ai-auto-register/server/internal/model"
	"gorm.io/gorm"
)

type ProxyResource struct {
	mu             sync.RWMutex
	proxies        []model.Proxy
	groupedByID    map[uint][]model.Proxy
	groupedByName  map[string][]model.Proxy
	db             *gorm.DB
	index          int
	groupIDIndexes map[uint]int
	groupNameIndex map[string]int
}

func NewProxyResource(db *gorm.DB) *ProxyResource {
	r := &ProxyResource{db: db}
	r.Reload()
	return r
}

func (r *ProxyResource) Reload() {
	var proxies []model.Proxy
	r.db.Preload("ProxyGroup").Where("status = ?", "active").Find(&proxies)
	groupedByID := make(map[uint][]model.Proxy)
	groupedByName := make(map[string][]model.Proxy)
	for _, proxy := range proxies {
		if proxy.ProxyGroupID != nil {
			groupedByID[*proxy.ProxyGroupID] = append(groupedByID[*proxy.ProxyGroupID], proxy)
		}
		if proxy.ProxyGroup != nil {
			group := strings.TrimSpace(proxy.ProxyGroup.Name)
			if group != "" {
				groupedByName[group] = append(groupedByName[group], proxy)
			}
		}
	}
	r.mu.Lock()
	r.proxies = proxies
	r.groupedByID = groupedByID
	r.groupedByName = groupedByName
	r.index = 0
	r.groupIDIndexes = make(map[uint]int)
	r.groupNameIndex = make(map[string]int)
	r.mu.Unlock()
}

func (r *ProxyResource) Next() *model.Proxy {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.proxies) == 0 {
		return nil
	}
	p := r.proxies[r.index%len(r.proxies)]
	r.index++
	return &p
}

func (r *ProxyResource) NextByGroupID(groupID uint) *model.Proxy {
	r.mu.Lock()
	defer r.mu.Unlock()
	matched := r.groupedByID[groupID]
	if len(matched) == 0 {
		return nil
	}
	idx := r.groupIDIndexes[groupID] % len(matched)
	r.groupIDIndexes[groupID] = idx + 1
	p := matched[idx]
	return &p
}

func (r *ProxyResource) NextByGroupName(group string) *model.Proxy {
	group = strings.TrimSpace(group)
	r.mu.Lock()
	defer r.mu.Unlock()
	matched := r.groupedByName[group]
	if len(matched) == 0 {
		return nil
	}
	idx := r.groupNameIndex[group] % len(matched)
	r.groupNameIndex[group] = idx + 1
	p := matched[idx]
	return &p
}

func (r *ProxyResource) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.proxies)
}
