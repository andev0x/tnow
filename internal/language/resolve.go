package language

import (
	"sort"
	"strings"
	"sync"
)

// Suggestion describes a supported language in the /to autocomplete list.
type Suggestion struct {
	Language Language
	Name     string
	Region   string
}

type aliasEntry struct {
	key  string
	lang Language
}

var regionNames = map[Language]string{
	EN:   "United States",
	ES:   "Spain",
	FR:   "France",
	DE:   "Germany",
	ZH:   "China",
	ZHTW: "Taiwan",
	JA:   "Japan",
	KO:   "South Korea",
	PT:   "Portugal",
	RU:   "Russia",
	VI:   "Vietnam",
	HI:   "India",
	AR:   "Saudi Arabia",
	TH:   "Thailand",
	YUE:  "Hong Kong",
	ID:   "Indonesia",
	MS:   "Malaysia",
	MY:   "Myanmar",
	KM:   "Cambodia",
	LO:   "Laos",
	IT:   "Italy",
	NL:   "Netherlands",
	CS:   "Czechia",
	HR:   "Croatia",
	HU:   "Hungary",
	IS:   "Iceland",
	GA:   "Ireland",
	EU:   "Basque Country",
	GL:   "Galicia",
	SW:   "Tanzania",
	HT:   "Haiti",
	UZ:   "Uzbekistan",
	KK:   "Kazakhstan",
	KY:   "Kyrgyzstan",
}

// extraAliases contains language names, endonyms, country names and country
// codes that should resolve to a supported language. Language codes and display
// names are added automatically.
var extraAliases = map[Language][]string{
	EN: {
		"eng", "english", "united states", "united states of america", "usa", "us",
		"america", "american", "uk", "gb", "great britain", "britain", "british",
		"england", "australia", "au", "canada", "ca", "new zealand", "nz",
		"singapore", "sg", "hoa kỳ", "hoa ky", "mỹ", "anh", "úc", "uc",
	},
	ES: {
		"spa", "spanish", "español", "espanol", "spain", "españa", "espana",
		"mexico", "mexican", "mx", "argentina", "colombia", "co", "chile", "cl",
		"peru", "pe", "venezuela", "uruguay", "paraguay", "bolivia", "ecuador",
		"cuba", "dominican republic", "guatemala", "honduras", "el salvador",
		"nicaragua", "costa rica", "panama", "puerto rico", "tây ban nha",
		"tay ban nha", "chi lê", "chi le",
	},
	FR: {
		"fra", "fre", "french", "français", "francais", "france", "monaco",
		"senegal", "ivory coast", "côte d'ivoire", "cote d'ivoire", "cameroon",
		"madagascar", "pháp", "phap",
	},
	DE: {
		"deu", "ger", "german", "deutsch", "deutschland", "germany", "austria",
		"at", "switzerland", "liechtenstein", "luxembourg", "đức", "duc",
	},
	ZH: {
		"zho", "chi", "chinese", "mandarin", "中文", "汉语", "漢語", "普通话",
		"普通話", "china", "mainland china", "prc", "cn", "trung quốc",
		"trung quoc", "trung hoa",
	},
	ZHTW: {
		"zh-tw", "zh_tw", "zh-hant", "zhtw", "traditional chinese",
		"chinese traditional", "taiwan", "taiwanese", "tw", "繁體中文",
		"繁体中文", "台湾", "臺灣", "đài loan", "dai loan",
	},
	JA: {
		"jpn", "japanese", "日本", "日本語", "japan", "jp", "nhật bản",
		"nhat ban", "nhật", "nhat",
	},
	KO: {
		"kor", "korean", "한국어", "한국", "south korea", "korea",
		"republic of korea", "north korea", "kr", "hàn quốc", "han quoc",
	},
	PT: {
		"por", "portuguese", "português", "portugues", "portugal", "brazil",
		"brasil", "br", "angola", "mozambique", "cape verde", "bồ đào nha",
		"bo dao nha", "braxin",
	},
	RU: {
		"rus", "russian", "русский", "russia", "ru", "nga",
	},
	VI: {
		"vie", "vietnamese", "tiếng việt", "tieng viet", "việt nam",
		"viet nam", "vietnam", "vn",
	},
	HI: {
		"hin", "hindi", "हिन्दी", "हिंदी", "india", "indian", "in", "ấn độ",
		"an do",
	},
	AR: {
		"ara", "arabic", "العربية", "saudi arabia", "sa", "uae",
		"united arab emirates", "ae", "egypt", "eg", "qatar", "qa", "kuwait",
		"kw", "iraq", "iq", "jordan", "jo", "lebanon", "lb", "syria", "sy",
		"yemen", "ye", "oman", "om", "bahrain", "bh", "algeria", "dz", "morocco",
		"ma", "tunisia", "tn", "libya", "ly", "sudan", "sd", "ả rập", "a rập",
		"ai cập", "ai cap", "a rập xê út", "a rap xe ut",
	},
	TH: {
		"tha", "thai", "ไทย", "thailand", "th", "thái lan", "thai lan",
	},
	YUE: {
		"cantonese", "粵語", "粤语", "hong kong", "hongkong", "hk", "macau",
		"macao", "mo", "quảng đông", "quang dong",
	},
	ID: {
		"ind", "indonesian", "bahasa indonesia", "indonesia", "id",
	},
	MS: {
		"msa", "may", "malay", "bahasa melayu", "malaysia", "mã lai", "ma lai",
	},
	MY: {
		"mya", "bur", "burmese", "myanmar", "burma", "my", "miến điện",
		"mien dien",
	},
	KM: {
		"khm", "khmer", "cambodia", "cambodian", "kh", "campuchia", "cao miên",
		"cao mien",
	},
	LO: {
		"lao", "laotian", "laos", "la",
	},
	IT: {
		"ita", "italian", "italiano", "italy", "it", "ý",
	},
	NL: {
		"nld", "dut", "dutch", "nederlands", "netherlands", "holland", "nl",
		"belgium", "be", "hà lan", "ha lan",
	},
	CS: {
		"ces", "cze", "czech", "čeština", "cestina", "czechia",
		"czech republic", "cz", "séc", "sec",
	},
	HR: {
		"hrv", "croatian", "hrvatski", "croatia", "hr",
	},
	HU: {
		"hun", "hungarian", "magyar", "hungary", "hu", "hung ga ri",
	},
	IS: {
		"isl", "icelandic", "íslenska", "islenska", "iceland", "is",
		"băng đảo", "bang dao",
	},
	GA: {
		"gle", "irish", "gaeilge", "ireland", "ie", "ai len",
	},
	EU: {
		"eus", "baq", "basque", "euskara", "basque country",
	},
	GL: {
		"glg", "galician", "galego", "galicia",
	},
	SW: {
		"swa", "swahili", "kiswahili", "tanzania", "kenya", "east africa",
	},
	HT: {
		"hat", "haitian", "haitian creole", "kreyòl", "kreyol", "haiti", "ht",
	},
	UZ: {
		"uzb", "uzbek", "o'zbek", "ozbek", "uzbekistan", "uz",
	},
	KK: {
		"kaz", "kazakh", "қазақ", "kazakhstan", "kz", "kk",
	},
	KY: {
		"kir", "kyrgyz", "кыргыз", "kyrgyzstan", "kirghizia", "kg", "ky",
	},
}

var (
	aliasOnce  sync.Once
	aliasIndex []aliasEntry
)

func aliases() []aliasEntry {
	aliasOnce.Do(func() {
		aliasIndex = buildAliasIndex()
	})
	return aliasIndex
}

func buildAliasIndex() []aliasEntry {
	entries := make([]aliasEntry, 0, 256)
	seen := make(map[string]bool)
	add := func(lang Language, aliases ...string) {
		for _, alias := range aliases {
			key := normalizeAlias(alias)
			if key == "" {
				continue
			}
			id := string(lang) + "\x00" + key
			if seen[id] {
				continue
			}
			seen[id] = true
			entries = append(entries, aliasEntry{key: key, lang: lang})
		}
	}

	for _, lang := range All() {
		add(lang, string(lang), DisplayName(lang))
		add(lang, extraAliases[lang]...)
	}
	add(Auto, "auto", "automatic", "detect", "detection")

	return entries
}

// normalizeAlias prepares user input and aliases for case-insensitive,
// whitespace-insensitive comparison. Hyphens and underscores are treated as
// spaces, so "zh-tw", "zh_tw" and "zh tw" all match.
func normalizeAlias(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Trim(s, `"'`)
	s = strings.NewReplacer(
		"_", " ", "-", " ",
		"'", " ", "’", " ", "`", " ",
		".", " ", ",", " ",
	).Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func matchScore(alias, query string) int {
	if query == "" {
		return 0
	}
	switch {
	case alias == query:
		return 0
	case strings.HasPrefix(alias, query):
		return 1
	case strings.Contains(alias, query):
		return 2
	default:
		return -1
	}
}

// Suggest returns supported languages matching a code, language name, country
// name, endonym, or country code. An empty query returns every supported
// language. Results are ranked exact first, then prefix, then substring.
func Suggest(query string) []Suggestion {
	q := normalizeAlias(query)
	best := make(map[Language]int)
	if q == "" {
		for _, lang := range All() {
			best[lang] = 0
		}
	} else {
		for _, entry := range aliases() {
			if entry.lang == Auto {
				continue
			}
			score := matchScore(entry.key, q)
			if score < 0 {
				continue
			}
			if previous, ok := best[entry.lang]; !ok || score < previous {
				best[entry.lang] = score
			}
		}
	}

	order := make(map[Language]int, len(All()))
	for i, lang := range All() {
		order[lang] = i
	}
	langs := make([]Language, 0, len(best))
	for lang := range best {
		langs = append(langs, lang)
	}
	sort.SliceStable(langs, func(i, j int) bool {
		left, right := best[langs[i]], best[langs[j]]
		if left != right {
			return left < right
		}
		return order[langs[i]] < order[langs[j]]
	})

	out := make([]Suggestion, 0, len(langs))
	for _, lang := range langs {
		out = append(out, Suggestion{
			Language: lang,
			Name:     DisplayName(lang),
			Region:   regionNames[lang],
		})
	}
	return out
}

// Resolve converts a language code, language name, country name, endonym, or
// country code into a supported language. Exact matches win; otherwise a unique
// prefix or (for queries of at least three characters) a unique substring is
// accepted. Ambiguous input is reported as unresolved so the UI can ask the
// user to choose.
func Resolve(input string) (Language, bool) {
	key := normalizeAlias(input)
	if key == "" {
		return Auto, false
	}

	exact := make(map[Language]bool)
	for _, entry := range aliases() {
		if entry.key == key {
			exact[entry.lang] = true
		}
	}
	if len(exact) == 1 {
		for lang := range exact {
			return lang, true
		}
	}
	if len(exact) > 1 {
		return Auto, false
	}

	prefix := make(map[Language]bool)
	for _, entry := range aliases() {
		if strings.HasPrefix(entry.key, key) {
			prefix[entry.lang] = true
		}
	}
	if len(prefix) == 1 {
		for lang := range prefix {
			return lang, true
		}
	}
	if len(prefix) > 1 {
		return Auto, false
	}

	if len([]rune(key)) >= 3 {
		substring := make(map[Language]bool)
		for _, entry := range aliases() {
			if strings.Contains(entry.key, key) {
				substring[entry.lang] = true
			}
		}
		if len(substring) == 1 {
			for lang := range substring {
				return lang, true
			}
		}
	}

	// Accept BCP-47-style tags such as en-US, pt-BR, vi-VN and zh-Hans.
	if lang, ok := resolveLanguageTag(input); ok {
		return lang, true
	}

	return Auto, false
}

// resolveLanguageTag resolves tags whose primary subtag is a supported
// language code. Region and script subtags are ignored, so en-US, en-GB and
// en-AU all resolve to English.
func resolveLanguageTag(input string) (Language, bool) {
	raw := strings.ToLower(strings.TrimSpace(input))
	raw = strings.ReplaceAll(raw, "_", "-")
	parts := strings.Split(raw, "-")
	if len(parts) < 2 {
		return Auto, false
	}
	primary := parts[0]
	for _, lang := range All() {
		if string(lang) == primary {
			return lang, true
		}
	}
	return Auto, false
}
