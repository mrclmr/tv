package main

var aliases = map[string]string{
	"arte":           "arte-de",
	"deutsche-welle": "dw",
}

func resolveAlias(title string, channels []channel) string {
	target, ok := aliases[title]
	if !ok {
		return title
	}
	_, found := channelToURL(target, channels)
	if !found {
		return title
	}
	return target
}
