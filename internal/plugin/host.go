package plugin

import (
	"encoding/json"
	"fmt"
)

// HostCall is the CGO host RPC. main sets this during init.
var HostCall func(method string, payload []byte) (response []byte, code int, err error)

func callHost(method string, payload any) (json.RawMessage, error) {
	if HostCall == nil {
		return nil, fmt.Errorf("host callback %s: not configured", method)
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal host callback %s: %w", method, err)
	}
	rawResponse, callCode, err := HostCall(method, rawPayload)
	if err != nil {
		return nil, err
	}
	if len(rawResponse) == 0 {
		return nil, fmt.Errorf("host callback %s returned no response, code=%d", method, callCode)
	}
	var env envelope
	if err := json.Unmarshal(rawResponse, &env); err != nil {
		return nil, fmt.Errorf("decode host envelope %s: %w", method, err)
	}
	if !env.OK {
		if env.Error != nil {
			return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		return nil, fmt.Errorf("host callback %s failed", method)
	}
	if callCode != 0 {
		return nil, fmt.Errorf("host callback %s returned code=%d", method, callCode)
	}
	return append(json.RawMessage(nil), env.Result...), nil
}
