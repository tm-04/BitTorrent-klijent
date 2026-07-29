package bencode

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func ParseInteger(s string) (int, int, error) {
	if len(s) == 0 || s[0] != 'i' {
		return 0, 0, errors.New("integer mora poceti s 'i'")
	}

	end := strings.IndexByte(s, 'e')
	if end == -1 {
		return 0, 0, errors.New("integer mora zavrsavati s 'e'")
	}

	value, err := strconv.Atoi(s[1:end])

	if err != nil {
		return 0, 0, fmt.Errorf("Neispravan broj")
	}

	return value, end + 1, nil
}

func ParseString(s string) (string, int, error) {
	colonIndex := strings.IndexByte(s, ':')
	if colonIndex == -1 {
		return "", 0, errors.New("nedostaje ':'")
	}

	length, err := strconv.Atoi(s[0:colonIndex])

	if err != nil {
		return "", 0, fmt.Errorf("neispravna duljina stringa")
	}

	stringStart := colonIndex + 1
	stringEnd := colonIndex + length + 1
	if stringEnd > len(s) {
		return "", 0, errors.New("okazivač premašuje duljinu stringa")
	}

	resultString := s[stringStart:stringEnd]

	end := stringEnd

	return resultString, end, nil
}

func ParseList(s string) ([]interface{}, int, error) {
	if len(s) == 0 || s[0] != 'l' {
		return nil, 0, errors.New("struktura nije lista")
	}

	index := 1

	var list []interface{}

	for index < len(s) {
		if s[index] == 'e' {
			index++
			return list, index, nil
		}

		value, consumed, err := Parse(s[index:])
		if err != nil {
			return nil, 0, fmt.Errorf("greska unutar liste: %w", err)
		}

		list = append(list, value)
		index += consumed

	}

	return nil, 0, errors.New("nedostaje 'e' na kraju liste")
}

func ParseDict(s string) (map[string]interface{}, int, error) {
	if len(s) == 0 || s[0] != 'd' {
		return nil, 0, errors.New("struktura nije dictionary")
	}

	index := 1

	dict := make(map[string]interface{})

	for index < len(s) {
		if s[index] == 'e' {
			index++
			return dict, index, nil
		}

		key, consumed, err := ParseString(s[index:])
		if err != nil {
			return nil, 0, fmt.Errorf("greska pri parsiranju kljuca: %w", err)
		}
		index += consumed

		value, consumed, err := Parse(s[index:])
		if err != nil {
			return nil, 0, fmt.Errorf("greska pri parsiranju vrijednosti: %w", err)
		}
		index += consumed

		dict[key] = value
	}

	return nil, 0, errors.New("nedostaje 'e' na kraju dicta")
}

func Parse(s string) (interface{}, int, error) {
	if len(s) == 0 {
		return nil, 0, errors.New("greska: prazan string")
	}

	switch {
	case s[0] == 'l':
		return ParseList(s)
	case s[0] == 'i':
		return ParseInteger(s)
	case s[0] == 'd':
		return ParseDict(s)
	case s[0] >= '0' && s[0] <= '9':
		return ParseString(s)
	default:
		return nil, 0, errors.New("nepostojeci format")
	}
}

/*func main() {
	data, err := os.ReadFile("ubuntu-26.04-desktop-amd64.iso.torrent")
	if err != nil {
		fmt.Println("greska pri citanju fajla:", err)
		return
	}

	result, _, err := Parse(string(data))
	if err != nil {
		fmt.Println("greska pri parsiranju:", err)
		return
	}

	dict := result.(map[string]interface{})

	announce, ok := dict["announce"].(string)
	if ok {
		fmt.Println("Tracker URL:", announce)
	}

	info, ok := dict["info"].(map[string]interface{})
	if !ok {
		fmt.Println("greska: info dict ne postoji ili nije dict")
		return
	}

	name, ok := info["name"].(string)
	if ok {
		fmt.Println("Ime:", name)
	}

	pieceLength, ok := info["piece length"].(int)
	if ok {
		fmt.Println("Piece length:", pieceLength)
	}

	length, ok := info["length"].(int)
	if ok {
		fmt.Println("Duljina fajla:", length)
	} else {
		fmt.Println("nema 'length', vjerojatno multi-file torrent (provjeri 'files')")
	}

	pieces, ok := info["pieces"].(string)
	if ok {
		fmt.Println("Duljina 'pieces' polja (bajtovi):", len(pieces))
		fmt.Println("Broj komada (pieces/20):", len(pieces)/20)
	}
}*/
