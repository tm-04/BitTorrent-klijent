package torrent

import (
	"crypto/sha1"
	"errors"
	"os"
	"strings"
	"torrentclient/bencode"
)

type TorrentInfo struct {
	Announce     string
	AnnounceList []string
	Name         string
	PieceLength  int
	Pieces       string
	Length       int
	InfoHash     [20]byte
}

// ucitavanje torrent datoteke, parsiranje i izvlacenje kljucnih podataka
func LoadTorrent(path string) (*TorrentInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	rawData := string(data)

	result, _, err := bencode.Parse(rawData)

	if err != nil {
		return nil, errors.New("greska prilikom parsiranja podataka (vanjski dict)")
	}

	dict, ok := result.(map[string]interface{})
	if !ok {
		return nil, errors.New("vanjska struktura nije dictionary")
	}

	announce, ok := dict["announce"].(string)
	if !ok {
		return nil, errors.New("nedostaje announce")
	}

	announceList := extractAnnounceList(dict)

	//info - key; cijeli dict - value; sada vrijednost pod dict["info"] spremamo odvojeno da bi mogli razdvojiti njegove podatke. posto je value pod info cijeli dict sad cemo ga samo odvojiti
	info, ok := dict["info"].(map[string]interface{})
	if !ok {
		return nil, errors.New("nedostaje info dict")
	}

	name, _ := info["name"].(string)
	pieceLength, _ := info["piece length"].(int)
	pieces, _ := info["pieces"].(string)
	length, _ := info["length"].(int)

	infoHash, err := computeInfoHash(rawData)
	if err != nil {
		return nil, errors.New("greska prilikom racunanja infoHasha")
	}

	return &TorrentInfo{
		Announce:     announce,
		AnnounceList: announceList,
		Name:         name,
		PieceLength:  pieceLength,
		Pieces:       pieces,
		Length:       length,
		InfoHash:     infoHash,
	}, nil

}

// sha1 hashing info dicta
func computeInfoHash(rawData string) ([20]byte, error) {
	var empty [20]byte
	infoKeyPos := strings.Index(rawData, "4:info")
	if infoKeyPos == -1 {
		return empty, errors.New("nedostaje info polje u torrent datoteci")
	}

	startPos := infoKeyPos + len("4:info")
	_, infoLength, err := bencode.ParseDict(rawData[startPos:])

	if err != nil {
		return empty, errors.New("greska prilikom racunanja duljine info dicta")
	}

	endPos := startPos + infoLength
	rawInfo := rawData[startPos:endPos]
	hash := sha1.Sum([]byte(rawInfo))

	return hash, nil

}

func (t *TorrentInfo) PieceHashes() [][20]byte {
	numPieces := len(t.Pieces) / 20
	hashes := make([][20]byte, numPieces)

	for i := 0; i < numPieces; i++ {
		copy(hashes[i][:], t.Pieces[i*20:(i+1)*20])
	}

	return hashes
}

func extractAnnounceList(dict map[string]interface{}) []string {
	var result []string

	announceListRaw, ok := dict["announce-list"].([]interface{})
	if !ok {
		return result
	}

	for _, tier := range announceListRaw {
		urls, ok := tier.([]interface{})
		if !ok {
			continue
		}

		for _, url := range urls {
			s, ok := url.(string)
			if ok {
				result = append(result, s)
			}
		}
	}

	return result
}
