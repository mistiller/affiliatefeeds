package woocommerce

import "strconv"

func decodeJson(m map[string]interface{}) []string {
	values := make([]string, 0, len(m))
	for _, v := range m {
		switch vv := v.(type) {
		case map[string]interface{}:
			for _, value := range decodeJson(vv) {
				values = append(values, value)
			}
		case string:
			values = append(values, vv)
		case float64:
			values = append(values, strconv.FormatFloat(vv, 'f', -1, 64))
		case []interface{}:
			// Arrays aren't currently handled, since you haven't indicated that we should
			// and it's non-trivial to do so.
		case bool:
			values = append(values, strconv.FormatBool(vv))
		case nil:
			values = append(values, "nil")
		}
	}
	return values
}
