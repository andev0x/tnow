package language

import (
	"strings"
	"unicode"
)

// Detect attempts a cheap deterministic language guess.
// It returns the language and a confidence in [0,1].
// It never claims certainty for ambiguous input.
func Detect(text string) (Language, float64) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Auto, 0
	}
	var (
		latin, cjk, kana, hangul int
		arabic, devanagari, thai int
		khmer, lao, myanmar      int
		cyrillic                 int
		vietnameseDiacritics     int
		total                    int
	)
	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}
		total++
		switch {
		case r >= 0x3040 && r <= 0x30FF: // hiragana/katakana
			kana++
		case r >= 0x4E00 && r <= 0x9FFF: // CJK unified
			cjk++
		case r >= 0xAC00 && r <= 0xD7AF: // hangul
			hangul++
		case r >= 0x0600 && r <= 0x06FF || r >= 0x0750 && r <= 0x077F || r >= 0x08A0 && r <= 0x08FF || r >= 0xFB50 && r <= 0xFDFF || r >= 0xFE70 && r <= 0xFEFF: // arabic
			arabic++
		case r >= 0x0900 && r <= 0x097F: // devanagari
			devanagari++
		case r >= 0x0E00 && r <= 0x0E7F: // thai
			thai++
		case r >= 0x1780 && r <= 0x17FF: // khmer
			khmer++
		case r >= 0x0E80 && r <= 0x0EFF: // lao
			lao++
		case r >= 0x1000 && r <= 0x109F: // myanmar
			myanmar++
		case r >= 0x0400 && r <= 0x04FF: // cyrillic
			cyrillic++
		default:
			latin++
			if isVietnameseDiacritic(r) {
				vietnameseDiacritics++
			}
		}
	}
	if total == 0 {
		return Auto, 0
	}
	if kana > 0 {
		return JA, float64(kana) / float64(total)
	}
	if hangul > 0 {
		return KO, float64(hangul) / float64(total)
	}
	if cjk > 0 && kana == 0 && hangul == 0 {
		// Could be zh or ja-kanji-only; lean zh with lower confidence.
		if cjk == total {
			return ZH, 0.7
		}
	}
	if arabic > 0 {
		conf := float64(arabic) / float64(total)
		if conf > 1 {
			conf = 1
		}
		return AR, conf
	}
	if devanagari > 0 {
		conf := float64(devanagari) / float64(total)
		if conf > 1 {
			conf = 1
		}
		return HI, conf
	}
	if thai > 0 {
		conf := float64(thai) / float64(total)
		if conf > 1 {
			conf = 1
		}
		return TH, conf
	}
	if khmer > 0 {
		conf := float64(khmer) / float64(total)
		if conf > 1 {
			conf = 1
		}
		return KM, conf
	}
	if lao > 0 {
		conf := float64(lao) / float64(total)
		if conf > 1 {
			conf = 1
		}
		return LO, conf
	}
	if myanmar > 0 {
		conf := float64(myanmar) / float64(total)
		if conf > 1 {
			conf = 1
		}
		return MY, conf
	}
	if cyrillic > 0 {
		conf := float64(cyrillic) / float64(total)
		if conf > 1 {
			conf = 1
		}
		return RU, conf
	}
	if vietnameseDiacritics > 0 {
		conf := float64(vietnameseDiacritics) / float64(latin)
		if conf > 1 {
			conf = 1
		}
		return VI, conf
	}
	return EN, 0.5
}

func isVietnameseDiacritic(r rune) bool {
	// Common Vietnamese diacritic letters.
	switch r {
	case 'à', 'á', 'ả', 'ã', 'ạ', 'ă', 'ắ', 'ằ', 'ẳ', 'ẵ', 'ặ',
		'â', 'ấ', 'ầ', 'ẩ', 'ẫ', 'ậ', 'è', 'é', 'ẻ', 'ẽ', 'ẹ',
		'ê', 'ế', 'ề', 'ể', 'ễ', 'ệ', 'ì', 'í', 'ỉ', 'ĩ', 'ị',
		'ò', 'ó', 'ỏ', 'õ', 'ọ', 'ô', 'ố', 'ồ', 'ổ', 'ỗ', 'ộ',
		'ơ', 'ớ', 'ờ', 'ở', 'ỡ', 'ợ', 'ù', 'ú', 'ủ', 'ũ', 'ụ',
		'ư', 'ứ', 'ừ', 'ử', 'ữ', 'ự', 'ỳ', 'ý', 'ỷ', 'ỹ', 'ỵ',
		'đ', 'À', 'Á', 'Ả', 'Ã', 'Ạ', 'Ă', 'Ắ', 'Ằ', 'Ẳ', 'Ẵ',
		'Ặ', 'Â', 'Ấ', 'Ầ', 'Ẩ', 'Ẫ', 'Ậ', 'È', 'É', 'Ẻ', 'Ẽ',
		'Ẹ', 'Ê', 'Ế', 'Ề', 'Ể', 'Ễ', 'Ệ', 'Ì', 'Í', 'Ỉ', 'Ĩ',
		'Ị', 'Ò', 'Ó', 'Ỏ', 'Õ', 'Ọ', 'Ô', 'Ố', 'Ồ', 'Ổ', 'Ỗ',
		'Ộ', 'Ơ', 'Ớ', 'Ờ', 'Ở', 'Ỡ', 'Ợ', 'Ù', 'Ú', 'Ủ', 'Ũ',
		'Ụ', 'Ư', 'Ứ', 'Ừ', 'Ử', 'Ữ', 'Ự', 'Ỳ', 'Ý', 'Ỷ', 'Ỹ', 'Ỵ', 'Đ':
		return true
	}
	return false
}
