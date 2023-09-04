package demo

var mapping = map[rune]rune{
	'a': 'R',
	'b': 'p',
	'c': 'V',
	'd': 'M',
	'e': 'K',
	'f': 'x',
	'g': 'k',
	'h': 'S',
	'i': 'Q',
	'j': 'L',
	'k': 'H',
	'l': 'h',
	'm': 'y',
	'n': 'a',
	'o': 'b',
	'p': 'e',
	'q': 'o',
	'r': 'd',
	's': 'G',
	't': 'E',
	'u': 'F',
	'v': 'C',
	'w': 's',
	'x': 'B',
	'y': 'g',
	'z': 'm',
	'A': 'I',
	'B': 'J',
	'C': 'i',
	'D': 'P',
	'E': 'X',
	'F': 'z',
	'G': 'j',
	'H': 'Y',
	'I': 'N',
	'J': 'T',
	'K': 'W',
	'L': 'r',
	'M': 'f',
	'N': 'Z',
	'O': 'D',
	'P': 't',
	'Q': 'A',
	'R': 'w',
	'S': 'l',
	'T': 'u',
	'U': 'q',
	'V': 'n',
	'W': 'v',
	'X': 'c',
	'Y': 'U',
	'Z': 'O',
	'0': '0',
	'1': '3',
	'2': '6',
	'3': '8',
	'4': '1',
	'5': '9',
	'6': '2',
	'7': '7',
	'8': '4',
	'9': '5',
}

func EncodeField(name string) string {
	value := []rune(name)
	for i, r := range value {
		if v, ok := mapping[r]; ok {
			value[i] = v
		}
	}
	return string(value)
}

func DecodeField(name string) string {
	value := []rune(name)
	for i, r := range value {
		for key, v := range mapping {
			if r == v {
				value[i] = key
			}
		}
	}
	return string(value)
}
