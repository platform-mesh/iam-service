package directive

import (
	"encoding/json"
	"testing"
)

func FuzzExtractResourceContext(f *testing.F) {
	f.Add(`{"context":{"group":"core.platform-mesh.io","kind":"Account","resource":{"name":"test"},"accountPath":"root:org"}}`)
	f.Add(`{"context":{"group":"","kind":"","resource":{"name":"","namespace":"ns"},"accountPath":""}}`)
	f.Add(`{"context":{}}`)
	f.Add(`{}`)
	f.Add(`{"context":"not-an-object"}`)
	f.Add(`{"wrongKey":true}`)
	f.Add(`not json`)
	f.Add(``)
	f.Add(`null`)
	f.Add(`{"context":null}`)

	f.Fuzz(func(t *testing.T, input string) {
		var args map[string]any
		if err := json.Unmarshal([]byte(input), &args); err != nil {
			return
		}
		// Must not panic regardless of map contents.
		_, _ = extractResourceContextFromArguments(args)
	})
}
