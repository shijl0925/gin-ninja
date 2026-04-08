// Package i18n provides locale negotiation and translation helpers for
// gin-ninja applications.
//
// # Locale negotiation
//
// NegotiateLocale parses an Accept-Language header and returns the best
// matching supported locale tag ("en" or "zh").  It falls back to "en" if no
// match is found.
//
// # Validation-error translation
//
// TranslateValidation converts a go-playground/validator tag (e.g. "required",
// "min", "email") into a human-readable message in the requested locale.
// Param is the constraint parameter value (e.g. "3" for min=3).
//
// # General message translation
//
// T translates a keyed message into the requested locale.  Arbitrary
// format arguments follow the key and are applied with fmt.Sprintf.  The
// call falls back to the English message if the key or locale is unknown.
//
// # Example
//
//	// In middleware/i18n.go the locale is stored in the gin context.
//	locale := i18n.NegotiateLocale(c.GetHeader("Accept-Language"))
//
//	// In binding.go the error is translated to the request locale.
//	msg := i18n.TranslateValidation("required", "", locale)  // "field is required"
package i18n

import (
	"fmt"
	"strings"

	"golang.org/x/text/language"
)

// Supported locale tags.
const (
	En = "en"
	Zh = "zh"
)

// Default is the locale used when no matching locale can be negotiated.
const Default = En

var supported = language.NewMatcher([]language.Tag{
	language.English,
	language.Chinese,
})

// NegotiateLocale parses an Accept-Language header value and returns the best
// matching supported locale ("en" or "zh").  Falls back to "en" when the
// header is empty or contains no recognisable tag.
func NegotiateLocale(acceptLanguage string) string {
	if strings.TrimSpace(acceptLanguage) == "" {
		return Default
	}
	tags, _, err := language.ParseAcceptLanguage(acceptLanguage)
	if err != nil || len(tags) == 0 {
		return Default
	}
	matched, _, _ := supported.Match(tags...)
	base, _ := matched.Base()
	switch base.String() {
	case "zh":
		return Zh
	default:
		return En
	}
}

// validationMessages is a two-level map: locale → validator-tag → message template.
// A single %s placeholder is expanded with the constraint param when present.
var validationMessages = map[string]map[string]string{
	En: {
		"required":  "field is required",
		"min":       "must be at least %s",
		"max":       "must be at most %s",
		"len":       "length must be exactly %s",
		"email":     "must be a valid email address",
		"url":       "must be a valid URL",
		"oneof":     "must be one of [%s]",
		"gt":        "must be greater than %s",
		"gte":       "must be greater than or equal to %s",
		"lt":        "must be less than %s",
		"lte":       "must be less than or equal to %s",
		"alphanum":  "must contain only letters and numbers",
		"alpha":     "must contain only letters",
		"numeric":   "must be a valid numeric value",
		"number":    "must be a valid number",
		"uuid":      "must be a valid UUID",
		"uuid4":     "must be a valid UUID v4",
		"ascii":     "must contain only ASCII characters",
		"lowercase": "must be lowercase",
		"uppercase": "must be uppercase",
		"ne":        "must not equal %s",
		"eq":        "must equal %s",
	},
	Zh: {
		"required":  "字段不能为空",
		"min":       "最小值为 %s",
		"max":       "最大值为 %s",
		"len":       "长度必须为 %s",
		"email":     "必须是有效的电子邮件地址",
		"url":       "必须是有效的 URL",
		"oneof":     "必须是 [%s] 之一",
		"gt":        "必须大于 %s",
		"gte":       "必须大于或等于 %s",
		"lt":        "必须小于 %s",
		"lte":       "必须小于或等于 %s",
		"alphanum":  "只能包含字母和数字",
		"alpha":     "只能包含字母",
		"numeric":   "必须是有效的数字",
		"number":    "必须是有效的数字",
		"uuid":      "必须是有效的 UUID",
		"uuid4":     "必须是有效的 UUID v4",
		"ascii":     "只能包含 ASCII 字符",
		"lowercase": "必须为小写",
		"uppercase": "必须为大写",
		"ne":        "不能等于 %s",
		"eq":        "必须等于 %s",
	},
}

// TranslateValidation returns a human-readable message for the given
// go-playground/validator tag in the requested locale.  param is the
// constraint parameter (e.g. "3" for min=3; empty string for tags without
// parameters such as "required").  Falls back to English when the locale or
// tag is not recognised.
func TranslateValidation(tag, param, locale string) string {
	msgs, ok := validationMessages[locale]
	if !ok {
		msgs = validationMessages[En]
	}
	tmpl, ok := msgs[tag]
	if !ok {
		// Fallback to English for unknown tags in non-English locales.
		if locale != En {
			if enMsgs, ok2 := validationMessages[En]; ok2 {
				tmpl, ok = enMsgs[tag]
			}
		}
		if !ok {
			return fmt.Sprintf("failed validation '%s'", tag)
		}
	}
	if strings.Contains(tmpl, "%s") {
		return fmt.Sprintf(tmpl, param)
	}
	return tmpl
}

// generalMessages is a two-level map: locale → message-key → message template.
var generalMessages = map[string]map[string]string{
	En: {
		"bad_request":  "bad request",
		"unauthorized": "unauthorized",
		"forbidden":    "forbidden",
		"not_found":    "not found",
		"conflict":     "conflict",
		"internal":     "internal server error",
		"timeout":      "request timed out",
		"validation":   "request validation failed",
		"rate_limited": "rate limit exceeded",
	},
	Zh: {
		"bad_request":  "请求有误",
		"unauthorized": "未授权",
		"forbidden":    "无权访问",
		"not_found":    "资源不存在",
		"conflict":     "资源冲突",
		"internal":     "服务器内部错误",
		"timeout":      "请求超时",
		"validation":   "请求参数校验失败",
		"rate_limited": "请求过于频繁，请稍后再试",
	},
}

// T returns the translated message for key in the given locale.  Variadic
// args are applied with fmt.Sprintf when the message contains format verbs.
// Falls back to the English message, then to the key itself, if nothing is
// found.
func T(locale, key string, args ...interface{}) string {
	msgs, ok := generalMessages[locale]
	if !ok {
		msgs = generalMessages[En]
	}
	msg, ok := msgs[key]
	if !ok {
		if locale != En {
			if enMsgs, ok2 := generalMessages[En]; ok2 {
				msg, ok = enMsgs[key]
			}
		}
		if !ok {
			return key
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}
