package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"torrentclient/peer"
	"torrentclient/torrent"
	"torrentclient/tracker"
)

type pieceResult struct {
	index int
	data  []byte
}

func worker() {

}

func main() {
	//ucitavanje torrenta
	t, err := torrent.LoadTorrent("LibreOffice_26.2.5_Win_x86-64.msi.torrent")
	if err != nil {
		fmt.Println("greska:", err)
		return
	}
	// test bencode parsera
	fmt.Println("Tracker URL:", t.Announce)
	fmt.Println("Ime:", t.Name)
	fmt.Println("Piece length:", t.PieceLength)
	fmt.Println("Duljina fajla:", t.Length)
	fmt.Println("Broj komada (pieces/20):", len(t.Pieces)/20)
	fmt.Printf("Info hash (hex): %x\n", t.InfoHash)
	fmt.Println("\nKraj testnog outputa\n----------------------------------")
	fmt.Printf("\n")
	// konekcija s trackerom i parse responsa
	peerList, peerID, err := tracker.SendGetParseResponse(t)
	if err != nil {
		log.Fatal("error")
	}
	fmt.Printf("\n\nPeerId: %s\n\n", peerID)

	tracker.PrintPeers(peerList)

	fmt.Println("\nkraj testnog ispisa za komunikaciju s trackerom\n----------------------------------")
	// TCP konekcija s peerom i handshake

	handshake, err := peer.BuildHandshake(t.InfoHash, peerID)
	if err != nil {
		fmt.Println("greska prilikom izrade handshakea")
	}

	index := 0
	var conn net.Conn
	for {
		address := fmt.Sprintf("%s:%s", peerList[index].IP, strconv.Itoa(int(peerList[index].Port)))
		conn, err = peer.ConnectToPeer(handshake, address)
		if err == nil && conn != nil {
			break
		}
		index++
		if index > 50 {
			break
		}
	}

	//conn := peer.ConnectToPeer(handshake, address)
	fmt.Println("kraj testnog ispisa za komunikaciju i handshake s peerom\n----------------------------------")

	//TREBA MALO UREDITI OVU POCETNU KOMUNIKACIJU ALI RADI ZA SAD
	buffer := make([]byte, 5)
	binary.BigEndian.PutUint32(buffer, 1)
	buffer[4] = peer.MsgInterested
	//ovo bi trebalo biti privremeno s ovim indexom i tim stvarima

	msg, err := peer.ReadMessage(conn)
	if err != nil {
		log.Fatal("greska", err)
	}

	fmt.Println("tip poruke", msg.ID)
	fmt.Println("Duljina payloada", len(msg.Payload))
	//fmt.Println("Payload", msg.Payload)

	n, err := conn.Write(buffer)
	if err != nil {
		log.Fatal(err)
	}
	if n != len(buffer) {
		log.Fatal("broj poslanih bitova nije jednak ocekivanoj duljini poruke")
	}

	for {
		msg, err = peer.ReadMessage(conn)
		if err != nil {
			log.Fatal(err)
		}

		if msg == nil {
			continue
		}

		if msg.ID == peer.MsgUnchoke {
			fmt.Println("Unchoke poruka primljena")
			break
		}
	}

	numPieces := len(t.Pieces) / 20
	fmt.Println("Broj pieceva:", numPieces)
	fmt.Println("Velicina jednog piecea", t.PieceLength)
	//fmt.Printf("Broj pieceva: %d")
	downloaded := make([]bool, numPieces)

	file, err := os.Create("torrent.msi")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	for i := 0; i < numPieces; i++ {
		var pieceLen int
		if i == numPieces-1 {
			pieceLen = t.Length - (i * t.PieceLength)
		} else {
			pieceLen = t.PieceLength
		}
		data, err := peer.DownloadPiece(t, conn, i, pieceLen)
		if err != nil {
			log.Fatal("Greska prilikom downloada", err)
		}

		_, err = file.WriteAt(data, int64(i)*int64(t.PieceLength))
		if err != nil {
			log.Fatal(err)
		}

		downloaded[i] = true

	}

}
