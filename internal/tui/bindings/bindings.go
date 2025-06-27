package bindings

import "github.com/charmbracelet/bubbles/key"

type Bindings interface {
	BindingKeys() []key.Binding
}

func KeyMapToSlice(km any) []key.Binding {
	switch km := km.(type) {
	case map[string]key.Binding:
		bindings := make([]key.Binding, 0, len(km))
		for _, v := range km {
			bindings = append(bindings, v)
		}
		return bindings
	default:
		return nil
	}
}
