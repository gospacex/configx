package thirdparty

type ThirdPartyConfig map[string]interface{}

func (t ThirdPartyConfig) GetServiceConfig(service string) (map[string]interface{}, bool) {
	if v, ok := t[service]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m, true
		}
	}
	return nil, false
}
