package tools

import (
	"net/http"

	"github.com/mitchellh/mapstructure"
)

func ParseURLParams(r *http.Request, out interface{}) error {
	query := r.URL.Query()

	flat := make(map[string]interface{})
	for k, v := range query {
		if len(v) > 0 {
			flat[k] = v[0]
		}
	}

	decoderConfig := &mapstructure.DecoderConfig{
		Result:           out,
		Squash:           true,
		WeaklyTypedInput: true,
		TagName:          "json",
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return err
	}

	return decoder.Decode(flat)

}
