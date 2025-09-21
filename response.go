package godaikin

import (
	"net/url"
	"strings"
)

// parseResponse parses a Daikin response string into a map
// Response format is like: "ret=OK,type=aircon,reg=eu,dst=1,ver=1_2_54"
func parseResponse(responseBody string) (map[string]string, error) {
	response := make(map[string]string)

	pairs := strings.Split(responseBody, ",")

	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		response[key] = value
	}

	ret, exists := response["ret"]
	if !exists {
		return nil, NewParseError("missing 'ret' field in response", nil)
	}

	if ret != "OK" {
		return make(map[string]string), nil
	}

	delete(response, "ret")

	if name, exists := response["name"]; exists {
		if decodedName, err := url.QueryUnescape(name); err == nil {
			response["name"] = decodedName
		}
	}

	return response, nil
}
