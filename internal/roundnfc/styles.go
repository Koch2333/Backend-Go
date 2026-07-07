package roundnfc

import "encoding/json"

var defaultBadgeStyleTemplates = []BadgeStyleTemplate{
	{Key: "sakura", Label: "樱花粉", Enabled: true, Payload: rawJSON(`{"theme":"sakura"}`)},
	{Key: "mint", Label: "薄荷绿", Enabled: true, Payload: rawJSON(`{"theme":"mint"}`)},
	{Key: "sky", Label: "天空蓝", Enabled: true, Payload: rawJSON(`{"theme":"sky"}`)},
	{Key: "lavender", Label: "薰衣草紫", Enabled: true, Payload: rawJSON(`{"theme":"lavender"}`)},
	{Key: "gold", Label: "香槟金", Enabled: true, Payload: rawJSON(`{"theme":"gold"}`)},
	{Key: "night", Label: "暗夜星", Enabled: true, Payload: rawJSON(`{"theme":"night"}`)},
}

func rawJSON(v string) json.RawMessage {
	return json.RawMessage(v)
}
