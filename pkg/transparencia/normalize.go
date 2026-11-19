package transparencia

import (
	"regexp"
	"strings"
	"unicode"
)

var nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9]+`)

// RemoveAccents remove acentos e diacríticos de strings em português para buscas rápidas.
func RemoveAccents(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 'á', 'à', 'ã', 'â', 'ä', 'Á', 'À', 'Ã', 'Â', 'Ä':
			b.WriteRune('a')
		case 'é', 'è', 'ê', 'ë', 'É', 'È', 'Ê', 'Ë':
			b.WriteRune('e')
		case 'í', 'ì', 'î', 'ï', 'Í', 'Ì', 'Î', 'Ï':
			b.WriteRune('i')
		case 'ó', 'ò', 'õ', 'ô', 'ö', 'Ó', 'Ò', 'Õ', 'Ô', 'Ö':
			b.WriteRune('o')
		case 'ú', 'ù', 'û', 'ü', 'Ú', 'Ù', 'Û', 'Ü':
			b.WriteRune('u')
		case 'ç', 'Ç':
			b.WriteRune('c')
		case 'ñ', 'Ñ':
			b.WriteRune('n')
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// Slugify transforma o nome do município em slug seguro para URLs (ex: "São Paulo" -> "sao-paulo").
func Slugify(s string) string {
	clean := RemoveAccents(s)
	clean = nonAlphaNumRegex.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-")
	return clean
}

// SlugifyCompact remove hífens do slug (ex: "Serra da Saudade" -> "serradasaudade").
func SlugifyCompact(s string) string {
	return strings.ReplaceAll(Slugify(s), "-", "")
}

// SlugifyNoPrepositions remove preposições comuns do nome antes de compactar (ex: "Serra da Saudade" -> "serrasaudade").
func SlugifyNoPrepositions(s string) string {
	tokens := Tokenize(s)
	filtered := make([]string, 0, len(tokens))
	for _, t := range tokens {
		switch t {
		case "da", "de", "do", "das", "dos", "e":
			continue
		default:
			filtered = append(filtered, t)
		}
	}
	if len(filtered) == 0 {
		return SlugifyCompact(s)
	}
	return strings.Join(filtered, "")
}

// Tokenize divide o termo em tokens normalizados.
func Tokenize(s string) []string {
	norm := RemoveAccents(s)
	parts := strings.Fields(norm)
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) > 0 {
			tokens = append(tokens, p)
		}
	}
	return tokens
}
