package ninja

import (
	"encoding/json"
	"sync"
)

type openAPICacheState struct {
	mu       sync.RWMutex
	main     []byte
	versions map[string][]byte
}

func (api *NinjaAPI) invalidateOpenAPICache() {
	api.openAPICache.mu.Lock()
	api.openAPICache.main = nil
	api.openAPICache.versions = map[string][]byte{}
	api.openAPICache.mu.Unlock()
}

func (api *NinjaAPI) openAPIBytes() ([]byte, error) {
	api.openAPICache.mu.RLock()
	cached := api.openAPICache.main
	api.openAPICache.mu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	built, err := json.Marshal(api.openAPI.build())
	if err != nil {
		return nil, err
	}

	api.openAPICache.mu.Lock()
	if api.openAPICache.main == nil {
		api.openAPICache.main = built
	} else {
		built = api.openAPICache.main
	}
	api.openAPICache.mu.Unlock()
	return built, nil
}

func (api *NinjaAPI) versionOpenAPIBytes(version string) ([]byte, bool, error) {
	api.openAPICache.mu.RLock()
	if cached, ok := api.openAPICache.versions[version]; ok {
		api.openAPICache.mu.RUnlock()
		return cached, true, nil
	}
	api.openAPICache.mu.RUnlock()

	spec, ok := api.lookupVersionSpec(version)
	if !ok {
		return nil, false, nil
	}
	built, err := json.Marshal(spec.build())
	if err != nil {
		return nil, true, err
	}

	api.openAPICache.mu.Lock()
	if cached, exists := api.openAPICache.versions[version]; exists {
		built = cached
	} else {
		if api.openAPICache.versions == nil {
			api.openAPICache.versions = map[string][]byte{}
		}
		api.openAPICache.versions[version] = built
	}
	api.openAPICache.mu.Unlock()
	return built, true, nil
}
