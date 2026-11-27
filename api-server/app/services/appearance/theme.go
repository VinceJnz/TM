package appearance

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Theme struct {
	Name       string `json:"name,omitempty"`
	Primary    string `json:"primary,omitempty"`
	Background string `json:"background,omitempty"`
	Surface    string `json:"surface,omitempty"`
	Text       string `json:"text,omitempty"`
	Font       string `json:"font,omitempty"`
	Logo       string `json:"logo,omitempty"`
	Icon       string `json:"icon,omitempty"`
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var fonts = map[string]string{
	"system": `system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`,
	"serif":  `Georgia, "Times New Roman", serif`,
	"sans":   `Arial, Helvetica, sans-serif`,
}

func (t Theme) WithDefaults() Theme {
	if t.Name == "" {
		t.Name = "Trip Manager"
	}
	if t.Primary == "" {
		t.Primary = "#2563eb"
	}
	if t.Background == "" {
		t.Background = "#f5f7fb"
	}
	if t.Surface == "" {
		t.Surface = "#ffffff"
	}
	if t.Text == "" {
		t.Text = "#1f2937"
	}
	if t.Font == "" {
		t.Font = "system"
	}
	return t
}

func (t Theme) Validate() error {
	t = t.WithDefaults()
	if strings.TrimSpace(t.Name) == "" || utf8.RuneCountInString(t.Name) > 80 || strings.IndexFunc(t.Name, unicode.IsControl) >= 0 {
		return errors.New("Site name must contain 1 to 80 printable characters")
	}
	for _, color := range []string{t.Primary, t.Background, t.Surface, t.Text} {
		if !hexColor.MatchString(color) {
			return errors.New("Use six-digit hex colors")
		}
	}
	if _, ok := fonts[t.Font]; !ok {
		return errors.New("Choose an available font")
	}
	for _, pair := range [][2]string{{t.Text, t.Background}, {t.Text, t.Surface}, {t.Primary, t.Background}, {t.Primary, t.Surface}, {t.Primary, "#ffffff"}} {
		if contrast(pair[0], pair[1]) < 4.5 {
			return errors.New("Choose colors with at least 4.5:1 contrast for text, links and buttons")
		}
	}
	if _, _, err := Image(t.Logo, 256*1024, 1024); err != nil {
		return errors.New("Logo must be a PNG or JPEG up to 256 KiB and 1024 pixels per side")
	}
	if _, _, err := Image(t.Icon, 32*1024, 256); err != nil {
		return errors.New("Browser icon must be a PNG or JPEG up to 32 KiB and 256 pixels per side")
	}
	return nil
}

func Image(encoded string, maxBytes, maxSide int) ([]byte, string, error) {
	if encoded == "" {
		return nil, "", nil
	}
	if len(encoded) > base64.StdEncoding.EncodedLen(maxBytes) {
		return nil, "", errors.New("image too large")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) > maxBytes {
		return nil, "", errors.New("invalid image encoding")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > maxSide || cfg.Height > maxSide {
		return nil, "", errors.New("invalid image dimensions or format")
	}
	if _, _, err = image.Decode(bytes.NewReader(data)); err != nil {
		return nil, "", err
	}
	return data, "image/" + format, nil
}

func contrast(a, b string) float64 {
	x, y := luminance(a), luminance(b)
	return (math.Max(x, y) + 0.05) / (math.Min(x, y) + 0.05)
}

func luminance(color string) float64 {
	coefficients := [...]float64{0.2126, 0.7152, 0.0722}
	var value float64
	for i, coefficient := range coefficients {
		channel, _ := strconv.ParseUint(color[1+2*i:3+2*i], 16, 8)
		s := float64(channel) / 255
		if s <= 0.04045 {
			s /= 12.92
		} else {
			s = math.Pow((s+0.055)/1.055, 2.4)
		}
		value += coefficient * s
	}
	return value
}

func (t Theme) CSS() string {
	t = t.WithDefaults()
	return ":root{--tm-font:" + fonts[t.Font] + ";--tm-primary:" + t.Primary + ";--tm-bg:" + t.Background + ";--tm-surface:" + t.Surface + ";--tm-text:" + t.Text +
		";--tm-primary-hover:color-mix(in srgb,var(--tm-primary),black 16%);--tm-selected-bg:color-mix(in srgb,var(--tm-primary) 10%,var(--tm-surface));--tm-selected-text:var(--tm-text);--tm-selected-border:color-mix(in srgb,var(--tm-primary) 22%,var(--tm-surface));--tm-highlight-bg:var(--tm-selected-bg);--tm-highlight-border:var(--tm-selected-border);--tm-highlight-ring:var(--tm-selected-border);--tm-action-shadow:0 10px 24px var(--tm-selected-border);--tm-overlay-surface:var(--tm-surface);--tm-surface-muted:color-mix(in srgb,var(--tm-text) 6%,var(--tm-surface));--tm-border:color-mix(in srgb,var(--tm-text) 22%,var(--tm-surface));--tm-text-subtle:var(--tm-text)}\n"
}
