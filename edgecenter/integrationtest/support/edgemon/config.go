package edgemon

func WithName(name string) map[string]interface{} {
	return map[string]interface{}{"name": name}
}

func WithReceiver(receiver string) map[string]interface{} {
	return map[string]interface{}{"receiver": receiver}
}

func Merge(base map[string]interface{}, parts ...map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for key, value := range base {
		out[key] = value
	}

	for _, part := range parts {
		for key, value := range part {
			out[key] = value
		}
	}

	return out
}
