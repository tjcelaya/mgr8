package mutil

import "strings"

type BinaryChangeIntent int

const (
	IntentNone = iota
	IntentAdd
	IntentRemove
	IntentFlip
)

func BuildCommaSeparatedQualifiedSymbolList(csvList string, symbolQualifier rune) [][]string {

	if !strings.ContainsRune(csvList, ',') {
		return [][]string{buildSymbolParts(csvList, symbolQualifier)}
	}

	syms := make([][]string, 0)

	for _, s := range strings.Split(csvList, ",") {
		if 0 == len(s) {
			continue
		}

		syms = append(syms, buildSymbolParts(s, symbolQualifier))

	}

	return syms
}

func buildSymbolParts(sym string, qualifier rune) []string {
	if strings.Contains(sym, string(qualifier)) {
		return strings.Split(sym, string(qualifier))
	} else {
		return []string{sym}
	}
}
