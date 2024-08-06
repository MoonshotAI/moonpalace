package merge

import "encoding/json"

type Merger struct {
	StreamFields []string
	IndexFields  []string
}

func (m *Merger) MergeObject(prev, next map[string]any) {
	for k, v := range next {
		switch vv := v.(type) {
		case map[string]any:
			if pv, ok := prev[k]; ok {
				if pvv, isObj := pv.(map[string]any); isObj {
					m.MergeObject(pvv, vv)
					continue
				}
			}
			prev[k] = vv
		case []any:
			if pv, ok := prev[k]; ok {
				if pvv, isArr := pv.([]any); isArr {
					m.MergeArray(&pvv, vv)
					prev[k] = pvv
					continue
				}
			}
			prev[k] = vv
		case string:
			if m.isStreamField(k) {
				if pv, ok := prev[k]; ok {
					if pvv, isStr := pv.(string); isStr {
						prev[k] = pvv + vv
						continue
					}
				}
				prev[k] = vv
			} else if vv != "" {
				prev[k] = vv
			}
		default:
			if v != nil {
				prev[k] = v
			}
		}
	}
}

func (m *Merger) MergeArray(prev *[]any, next []any) {
	for _, v := range next {
		if ov, isObj := v.(map[string]any); isObj {
			indexNum, found := m.findIndex(ov)
			if found {
				if indexI64, err := indexNum.Int64(); err == nil {
					index := int(indexI64)
					if len(*prev) < (index + 1) {
						for range (index + 1) - len(*prev) {
							*prev = append(*prev, make(map[string]any))
						}
					}
					item := (*prev)[index]
					if itemObj, isObj := item.(map[string]any); isObj {
						m.MergeObject(itemObj, ov)
					} else {
						(*prev)[index] = ov
					}
				}
			}
		} else {
			*prev = append(*prev, v)
		}
	}
}

func (m *Merger) isStreamField(field string) bool {
	for _, streamField := range m.StreamFields {
		if streamField == field {
			return true
		}
	}
	return false
}

func (m *Merger) findIndex(obj map[string]any) (json.Number, bool) {
	for k, v := range obj {
		if m.isIndexField(k) {
			if jsNum, isNum := v.(json.Number); isNum {
				return jsNum, true
			}
		}
	}
	return "", false
}

func (m *Merger) isIndexField(field string) bool {
	for _, indexField := range m.IndexFields {
		if indexField == field {
			return true
		}
	}
	return false
}
