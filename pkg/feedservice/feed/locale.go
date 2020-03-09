package feed

import (
	"fmt"
	"strings"
)

var (
	ValidLanguageToCountry = map[string][]string{
		"sv": []string{
			"SE",
		},
		"en": []string{
			"GB",
			"UK",
		},
	}
	ValidLocaleToCountry = map[string][]string{
		"sv_se": []string{
			"SE",
		},
	}
	ValidShortToLongLaguage = map[string]string{
		"en": "English",
		"sv": "Swedish",
		"de": "German",
		"pl": "Polish",
	}
)

// Locale is an object that contains the mapping for country codes and language names
// For example: SE - sv - sv_se
type Locale struct {
	TwoLetterCode string
	Language      string
	LongLanguage  string
	Locale        string

	initialized bool
}

func NewLocale(twoLetterCode, language, locale string) (l *Locale, err error) {
	l = &Locale{
		TwoLetterCode: twoLetterCode,
		Language:      language,
		Locale:        locale,
	}

	_, exists := ValidShortToLongLaguage[strings.ToLower(language)]
	if !exists {
		return l, fmt.Errorf("Couldn't find long language for %s", language)
	}
	l.LongLanguage = ValidShortToLongLaguage[strings.ToLower(language)]

	err = l.Validate()
	if err != nil {
		return l, err
	}

	return l, nil
}

func (l *Locale) Validate() error {
	var (
		exists bool
	)
	if anyEmpty(
		l.TwoLetterCode,
		l.Language,
		l.Locale,
	) {
		return fmt.Errorf("Locale incomplete -%v", l)
	}

	if len(l.Language) != 2 || len(l.TwoLetterCode) != 2 {
		return fmt.Errorf("Language / Country Code needs to be specified with two letters")
	}

	_, exists = ValidLanguageToCountry[l.Language]
	if !exists {
		return fmt.Errorf("Couldn't map language to country - %v", l)
	}

	_, exists = ValidLocaleToCountry[l.Locale]
	if !exists {
		return fmt.Errorf("Couldn't map locale to country - %v", l)
	}

	return nil
}

func anyEmpty(str ...string) bool {
	for i := range str {
		if str[i] == "" {
			return true
		}
	}
	return false
}
