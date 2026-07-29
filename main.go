package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
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

func ParsePeers(peers string) ([]Peer, error) {
	var peerList []Peer

	if len(peers)%6 != 0 {
		log.Fatal("Kriva duljina peera")
		return peerList, nil
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

func main() {
	t, err := torrent.LoadTorrent("ubuntu-26.04-desktop-amd64.iso.torrent")
	if err != nil {
		fmt.Println("greska:", err)
		return
	}

	fmt.Println("Tracker URL:", t.Announce)
	fmt.Println("Ime:", t.Name)
	fmt.Println("Piece length:", t.PieceLength)
	fmt.Println("Duljina fajla:", t.Length)
	fmt.Println("Broj komada (pieces/20):", len(t.Pieces)/20)
	fmt.Printf("Info hash (hex): %x\n", t.InfoHash)

	urlParsed, err := url.Parse(t.Announce)
	if err != nil {
		log.Fatal(err)
	}

	values := urlParsed.Query()
	peerID := [20]byte{'-', 'T', 'C', '0', '0', '0', '1', '-', 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
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
		log.Fatal(err)
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode > 299 {
		log.Fatalf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
	}

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n%s\n", body)

	result, _, err := bencode.Parse(string(body))
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	fmt.Println("Peers:")
	/*for i, peer := range peerList {
		fmt.Printf("%d. IP: %s, Port: %d\n", i+1, peer.IP, peer.Port)
	}*/

	for i := 0; i < len(peerList); i += 1 {
		fmt.Printf("%d. IP: %s, Port: %d\n", i+1, peerList[i].IP, peerList[i].Port)
	}
}
