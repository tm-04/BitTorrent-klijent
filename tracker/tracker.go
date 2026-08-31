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
	"strings"
	"torrentclient/bencode"
	"torrentclient/torrent"
)

type Peer struct {
	IP   string
	Port uint16
}

func GenerateRandomID() ([20]byte, error) {
	var peerID [20]byte

	prefix := []byte{'-', 'Z', 'R', '0', '0', '0', '1', '-'}
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

func SendGetParseResponse(t *torrent.TorrentInfo) ([]Peer, [20]byte, error) {
	var empty []Peer
	var emptyid [20]byte

	/*for {
	}*/
	trackerUrl := pickTracker(t)
	//fmt.Println("Tracker URL:", trackerUrl)
	if trackerUrl == "" {
		return nil, emptyid, errors.New("nema dostupnog http trackera")
	}

	urlParsed, err := url.Parse(trackerUrl)
	if err != nil {
		return empty, emptyid, errors.New("greška prilikom parsiranja ip adrese trackera")
	}

	values := urlParsed.Query()

	peerID, err := GenerateRandomID()
	if err != nil {
		return nil, emptyid, errors.New("greska pri generiranju random peerID-a")
	}

	values.Add("port", strconv.Itoa(6881))
	values.Add("peer_id", string(peerID[:]))
	values.Add("info_hash", string(t.InfoHash[:]))
	values.Add("uploaded", strconv.Itoa(0))
	values.Add("downloaded", strconv.Itoa(0))
	values.Add("left", strconv.Itoa(t.Length))
	values.Add("compact", strconv.Itoa(1))
	urlParsed.RawQuery = values.Encode()

	res, err := http.Get(urlParsed.String())
	if err != nil {
		return empty, emptyid, err
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode > 299 {
		return empty, emptyid, fmt.Errorf("tracker vratio status %d: %s", res.StatusCode, body)
	}

	if err != nil {
		return empty, emptyid, err
	}

	//fmt.Printf("\n%s\n", body)

	result, _, err := bencode.Parse(string(body))
	if err != nil {
		return empty, emptyid, err
	}

	dict := result.(map[string]interface{})

	//interval, _ := dict["interval"].(int)
	peers, _ := dict["peers"].(string)

	/*//len vraca duljinu u byteovima
	fmt.Println("Interval: ", interval)
	fmt.Println("Duljina peers stringa (bajtovi):", len(peers))
	fmt.Println("Duljina cijelog body-ja:", len(body))
	fmt.Println("Duljina peers stringa:", len(peers))
	fmt.Println("Broj peerova (peers/6):", len(peers)/6)*/

	peerList, err := ParsePeers(peers)
	if err != nil {
		return nil, emptyid, err
	}

	return peerList, peerID, nil
}

func PrintPeers(peerList []Peer) {
	for i := 0; i < len(peerList); i += 1 {
		fmt.Printf("%d. IP: %s, Port: %d\n", i+1, peerList[i].IP, peerList[i].Port)
	}
}

// bira se prvi tracker koji koristi http/https protokol, a ne UDP
func pickTracker(torrent *torrent.TorrentInfo) string {
	prefix := "http"

	if strings.HasPrefix(torrent.Announce, prefix) {
		return torrent.Announce
	}

	for _, url := range torrent.AnnounceList {
		if strings.HasPrefix(url, prefix) {
			return url
		}
	}

	return ""
}
