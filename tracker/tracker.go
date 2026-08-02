package tracker

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"torrentclient/bencode"
	"torrentclient/torrent"
)

type Peer struct {
	IP   string
	Port uint16
}

func GenerateRandomID() ([20]byte, error) {
	var peerID [20]byte

	prefix := []byte{'-', 'T', 'C', '0', '0', '0', '1', '-'}
	copy(peerID[:8], prefix)

	randomSuffix := make([]byte, 12)
	rand.Read(randomSuffix)
	copy(peerID[8:], randomSuffix)

	return peerID, nil
}

func ParsePeers(peers string) ([]Peer, error) {
	var peerList []Peer

	if len(peers)%6 != 0 {
		return nil, errors.New("kriva duljina peers stringa")
	}

	numPeers := len(peers) / 6
	for i := 0; i < numPeers; i += 1 {
		peerBytes := peers[i*6 : i*6+6]
		ipBytes := peerBytes[0:4]
		portBytes := peerBytes[4:6]
		ip := fmt.Sprintf("%d.%d.%d.%d", ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])
		port := binary.BigEndian.Uint16([]byte(portBytes))
		peer := Peer{IP: ip, Port: port}
		peerList = append(peerList, peer)
	}

	return peerList, nil
}

func SendGetParseResponse(t *torrent.TorrentInfo) ([]Peer, error) {
	var empty []Peer

	urlParsed, err := url.Parse(t.Announce)
	if err != nil {
		return empty, errors.New("greška prilikom parsiranja ip adrese trackera")
	}

	values := urlParsed.Query()

	peerID, err := GenerateRandomID()
	if err != nil {
		return nil, errors.New("greska pri generiranju random peerID-a")
	}
	//test
	//fmt.Printf("\nRandom generirani ID: %s \n---------------------------", peerID)
	// kraj testa

	values.Add("port", strconv.Itoa(6881))
	values.Add("peer_id", string(peerID[:]))
	values.Add("info_hash", string(t.InfoHash[:]))
	values.Add("uploaded", strconv.Itoa(0))
	values.Add("downloaded", strconv.Itoa(0))
	values.Add("left", strconv.Itoa(t.Length))
	values.Add("compact", strconv.Itoa(1))
	urlParsed.RawQuery = values.Encode()

	fmt.Println("Url nakon modifikacije je: \n", urlParsed.String())

	res, err := http.Get(urlParsed.String())
	if err != nil {
		return empty, err
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode > 299 {
		return empty, fmt.Errorf("tracker vratio status %d: %s", res.StatusCode, body)
	}

	if err != nil {
		return empty, err
	}

	fmt.Printf("\n%s\n", body)

	result, _, err := bencode.Parse(string(body))
	if err != nil {
		return empty, err
	}

	dict := result.(map[string]interface{})

	interval, _ := dict["interval"].(int)
	peers, _ := dict["peers"].(string)

	//len vraca duljinu u byteovima
	fmt.Println("Interval: ", interval)
	fmt.Println("Duljina peers stringa (bajtovi):", len(peers))
	fmt.Println("Duljina cijelog body-ja:", len(body))
	fmt.Println("Duljina peers stringa:", len(peers))
	fmt.Println("Broj peerova (peers/6):", len(peers)/6)

	peerList, err := ParsePeers(peers)
	if err != nil {
		return nil, err
	}

	return peerList, nil
}

func PrintPeers(peerList []Peer) {
	for i := 0; i < len(peerList); i += 1 {
		fmt.Printf("%d. IP: %s, Port: %d\n", i+1, peerList[i].IP, peerList[i].Port)
	}
}
