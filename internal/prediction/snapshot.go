package prediction

import "encoding/json"

// CloneState returns an independent snapshot of the persisted JSON model.
// State consists only of JSON-safe value records; it contains no resources.
func CloneState(state State) State {
	data, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	var cloned State
	if err := json.Unmarshal(data, &cloned); err != nil {
		panic(err)
	}
	return cloned
}
