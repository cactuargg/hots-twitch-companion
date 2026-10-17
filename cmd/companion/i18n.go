package main

import "hots-twitch-companion/internal/i18n"

// Transitional aliases keep all call sites and tests using the same messages.
const (
	langEn = i18n.En
	langRu = i18n.Ru
)

func normalizeLang(value string) string { return i18n.Normalize(value) }
func tr(lang, key string, args ...any) string { return i18n.Text(lang,key,args...) }
