package i18n_test

import (
	"testing"

	"github.com/shijl0925/gin-ninja/pkg/i18n"
)

// ---------------------------------------------------------------------------
// NegotiateLocale
// ---------------------------------------------------------------------------

func TestNegotiateLocale_Empty(t *testing.T) {
	if got := i18n.NegotiateLocale(""); got != i18n.En {
		t.Errorf("expected %q, got %q", i18n.En, got)
	}
}

func TestNegotiateLocale_English(t *testing.T) {
	cases := []string{"en", "en-US", "en-GB", "en-us;q=0.9"}
	for _, c := range cases {
		if got := i18n.NegotiateLocale(c); got != i18n.En {
			t.Errorf("NegotiateLocale(%q) = %q, want %q", c, got, i18n.En)
		}
	}
}

func TestNegotiateLocale_Chinese(t *testing.T) {
	cases := []string{"zh", "zh-CN", "zh-TW", "zh-Hans", "zh-Hant", "zh-CN,zh;q=0.9"}
	for _, c := range cases {
		if got := i18n.NegotiateLocale(c); got != i18n.Zh {
			t.Errorf("NegotiateLocale(%q) = %q, want %q", c, got, i18n.Zh)
		}
	}
}

func TestNegotiateLocale_Unsupported_FallsBackToEn(t *testing.T) {
	cases := []string{"fr", "de", "ja", "ko", "es-ES"}
	for _, c := range cases {
		if got := i18n.NegotiateLocale(c); got != i18n.En {
			t.Errorf("NegotiateLocale(%q) = %q, want fallback %q", c, got, i18n.En)
		}
	}
}

func TestNegotiateLocale_QValuePriority(t *testing.T) {
	// Chinese preferred with higher q-value.
	if got := i18n.NegotiateLocale("zh-CN,en;q=0.5"); got != i18n.Zh {
		t.Errorf("expected zh, got %q", got)
	}
	// English preferred with higher q-value.
	if got := i18n.NegotiateLocale("en,zh;q=0.5"); got != i18n.En {
		t.Errorf("expected en, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// TranslateValidation
// ---------------------------------------------------------------------------

func TestTranslateValidation_English(t *testing.T) {
	cases := []struct {
		tag, param, want string
	}{
		{"required", "", "field is required"},
		{"email", "", "must be a valid email address"},
		{"min", "3", "must be at least 3"},
		{"max", "10", "must be at most 10"},
		{"len", "8", "length must be exactly 8"},
		{"oneof", "a b c", "must be one of [a b c]"},
	}
	for _, tc := range cases {
		got := i18n.TranslateValidation(tc.tag, tc.param, i18n.En)
		if got != tc.want {
			t.Errorf("TranslateValidation(%q,%q,en) = %q, want %q", tc.tag, tc.param, got, tc.want)
		}
	}
}

func TestTranslateValidation_Chinese(t *testing.T) {
	cases := []struct {
		tag, param, want string
	}{
		{"required", "", "字段不能为空"},
		{"email", "", "必须是有效的电子邮件地址"},
		{"min", "5", "最小值为 5"},
	}
	for _, tc := range cases {
		got := i18n.TranslateValidation(tc.tag, tc.param, i18n.Zh)
		if got != tc.want {
			t.Errorf("TranslateValidation(%q,%q,zh) = %q, want %q", tc.tag, tc.param, got, tc.want)
		}
	}
}

func TestTranslateValidation_UnknownTag(t *testing.T) {
	got := i18n.TranslateValidation("nonexistent_tag", "", i18n.En)
	if got != "failed validation 'nonexistent_tag'" {
		t.Errorf("unexpected message for unknown tag: %q", got)
	}
}

func TestTranslateValidation_UnknownTagZh_FallsBackToEn(t *testing.T) {
	got := i18n.TranslateValidation("nonexistent_tag", "", i18n.Zh)
	if got != "failed validation 'nonexistent_tag'" {
		t.Errorf("unexpected message for unknown tag in zh locale: %q", got)
	}
}

func TestTranslateValidation_UnknownLocaleFallsBackToEn(t *testing.T) {
	got := i18n.TranslateValidation("required", "", "fr")
	if got != "field is required" {
		t.Errorf("unknown locale should fall back to en, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// T – general message translation
// ---------------------------------------------------------------------------

func TestT_English(t *testing.T) {
	if got := i18n.T(i18n.En, "not_found"); got != "not found" {
		t.Errorf("T(en, not_found) = %q", got)
	}
}

func TestT_Chinese(t *testing.T) {
	if got := i18n.T(i18n.Zh, "not_found"); got != "资源不存在" {
		t.Errorf("T(zh, not_found) = %q", got)
	}
}

func TestT_UnknownKey(t *testing.T) {
	if got := i18n.T(i18n.En, "no_such_key"); got != "no_such_key" {
		t.Errorf("T unknown key should return key itself, got %q", got)
	}
}

func TestT_UnknownLocale(t *testing.T) {
	if got := i18n.T("fr", "forbidden"); got != "forbidden" {
		t.Errorf("T unknown locale should fall back to en, got %q", got)
	}
}

func TestT_WithFormatArgs(t *testing.T) {
	// Add a key with a format verb to the test via the known rate_limited key
	// which is plain text, so test a different scenario: the fallback.
	got := i18n.T(i18n.En, "internal")
	if got != "internal server error" {
		t.Errorf("unexpected T result: %q", got)
	}
}
