package browser

import "agentium/internal/fingerprint"

func buildUserAgentData(profile fingerprint.Profile) map[string]any {
	if profile.UserAgentMetadata == nil {
		return map[string]any{}
	}

	brands := make([]map[string]string, 0, len(profile.UserAgentMetadata.Brands))
	for _, brand := range profile.UserAgentMetadata.Brands {
		if brand == nil {
			continue
		}
		brands = append(brands, map[string]string{
			"brand":   brand.Brand,
			"version": brand.Version,
		})
	}

	fullVersionList := make([]map[string]string, 0, len(profile.UserAgentMetadata.FullVersionList))
	for _, brand := range profile.UserAgentMetadata.FullVersionList {
		if brand == nil {
			continue
		}
		fullVersionList = append(fullVersionList, map[string]string{
			"brand":   brand.Brand,
			"version": brand.Version,
		})
	}

	return map[string]any{
		"brands":          brands,
		"mobile":          profile.UserAgentMetadata.Mobile,
		"platform":        profile.UserAgentMetadata.Platform,
		"architecture":    profile.UserAgentMetadata.Architecture,
		"bitness":         profile.UserAgentMetadata.Bitness,
		"model":           profile.UserAgentMetadata.Model,
		"platformVersion": profile.UserAgentMetadata.PlatformVersion,
		"uaFullVersion":   profile.UserAgentMetadata.FullVersion,
		"fullVersionList": fullVersionList,
	}
}

func firstLanguage(languages []string) string {
	if len(languages) == 0 {
		return "en-US"
	}
	return languages[0]
}
