package streamdeck

func showAlertMessage(context string) map[string]string {
	return map[string]string{"event": "showAlert", "context": context}
}

func showOkMessage(context string) map[string]string {
	return map[string]string{"event": "showOk", "context": context}
}

func (c *Client) SetTitle(context, title string) error {
	msg := map[string]any{
		"event":   "setTitle",
		"context": context,
		"payload": map[string]any{"title": title, "target": 0},
	}
	return c.writeJSON(msg)
}

func (c *Client) ShowAlert(context string) error {
	return c.writeJSON(showAlertMessage(context))
}

func (c *Client) ShowOk(context string) error {
	return c.writeJSON(showOkMessage(context))
}

func (c *Client) SetState(context string, state int) error {
	msg := map[string]any{
		"event":   "setState",
		"context": context,
		"payload": map[string]any{"state": state},
	}
	return c.writeJSON(msg)
}

func (c *Client) SetSettings(context string, settings any) error {
	msg := map[string]any{
		"event":   "setSettings",
		"context": context,
		"payload": settings,
	}
	return c.writeJSON(msg)
}

func (c *Client) SendToPropertyInspector(context, action string, payload any) error {
	msg := map[string]any{
		"event":   "sendToPropertyInspector",
		"context": context,
		"action":  action,
		"payload": payload,
	}
	return c.writeJSON(msg)
}
