package merge

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

var merger = &Merger{
	StreamFields: []string{"content", "arguments"},
	IndexFields:  []string{"index"},
}

func TestMerger_MergeObject(t *testing.T) {
	var (
		object = make(map[string]any)
		chunks = []string{
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"search:0","type":"function","function":{"name":"search","arguments":""}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\n"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"    "}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\""}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"query"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\""}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":":"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":" \""}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"Context"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"id":"search:0","type":"function","function":{"name":"search","arguments":""}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\n"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"    "}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\""}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"query"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\""}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":":"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":" \""}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"Context"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":" C"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"aching"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":" "}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"技术"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"\n"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"}"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"自然科学"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"}"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls","usage":{"prompt_tokens":146,"completion_tokens":26,"total_tokens":172}}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":" C"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"aching"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":" "}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"技术"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"\n"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"}"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"室外"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\n"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"}"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":1,"delta":{},"finish_reason":"tool_calls","usage":{"prompt_tokens":146,"completion_tokens":25,"total_tokens":171}}]}`,
		}
	)
	for _, chunk := range chunks {
		var next map[string]any
		decoder := json.NewDecoder(strings.NewReader(chunk))
		decoder.UseNumber()
		if err := decoder.Decode(&next); err != nil {
			t.Fatal(err)
		}
		merger.MergeObject(object, next)
	}
	const good = `{"id":"chatcmpl-7a799bb736944f95bc3b0880f8270f6b","object":"chat.completion.chunk","created":1722877271,"model":"moonshot-v1-128k","choices":[{"index":0,"delta":{"role":"assistant","content":"","tool_calls":[{"index":0,"id":"search:0","type":"function","function":{"name":"search","arguments":"{\n    \"query\": \"Context Caching 技术\"\n}自然科学}"}}]},"finish_reason":"tool_calls","usage":{"prompt_tokens":146,"completion_tokens":26,"total_tokens":172}},{"index":1,"delta":{"role":"assistant","content":"","tool_calls":[{"index":0,"id":"search:0","type":"function","function":{"name":"search","arguments":"{\n    \"query\": \"Context Caching 技术\"\n}室外\n}"}}]},"finish_reason":"tool_calls","usage":{"prompt_tokens":146,"completion_tokens":25,"total_tokens":171}}]}`
	var want = make(map[string]any)
	decoder := json.NewDecoder(strings.NewReader(good))
	decoder.UseNumber()
	if err := decoder.Decode(&want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, object) {
		wantjs, err := json.Marshal(want)
		if err != nil {
			t.Fatal(err)
		}
		objectjs, err := json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("\nwant: %s\ngot:  %s", string(wantjs), string(objectjs))
	}
}
