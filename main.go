package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"torrentclient/peer"
	"torrentclient/torrent"
	"torrentclient/tracker"
)

type pieceResult struct {
	index int
	data  []byte
}

func worker(t *torrent.TorrentInfo, handshake []byte, peerPool chan tracker.Peer, pieceJobs chan int,
	results chan pieceResult, numPieces int, wg *sync.WaitGroup /*counter *int*/) {
	defer wg.Done()
	for {

		conn := peer.FindWorkingConnection(peerPool, handshake)
		if conn == nil {
			fmt.Println("Worker nema dostupnih peerova")
			return
		}

		lostConnection := false

		for index := range pieceJobs {
			var pieceLen int
			if index == numPieces-1 {
				pieceLen = t.Length - (index * t.PieceLength)
			} else {
				pieceLen = t.PieceLength
			}

			data, err := peer.DownloadPiece(t, conn, index, pieceLen)
			if err != nil {
				//fmt.Println("worker izgubio konekciju na piecu:", index, ":", err)
				pieceJobs <- index
				lostConnection = true
				break
			}

			results <- pieceResult{index: index, data: data}

		}

		conn.Close()
		if !lostConnection {
			return
		}
	}

}

func main() {
	//ucitavanje torrenta
	if len(os.Args) < 2 {
		log.Fatal("Za pokretanje aplikacije potrebno je nakon run komande kao argument proslijediti ime filea\nPrimjer: go run main.go file.torrent\n")
	}
	t, err := torrent.LoadTorrent(os.Args[1])
	if err != nil {
		fmt.Println("greska:", err)
		return
	}

	// konekcija s trackerom i parse responsa
	peerList, peerID, err := tracker.SendGetParseResponse(t)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Ime:", t.Name)
	fmt.Println("Piece length:", t.PieceLength)
	fmt.Println("Duljina fajla:", t.Length)
	fmt.Println("Broj komada (pieces/20):", len(t.Pieces)/20)
	fmt.Printf("Info hash (hex): %x\n", t.InfoHash)
	fmt.Println("\nKraj testnog outputa\n----------------------------------")

	fmt.Printf("PeerId: %s\n", peerID)

	//tracker.PrintPeers(peerList)

	if len(peerList) < 1 {
		log.Fatal("Trazeni torrent file nema ni jednog peera, preuzimanje nije moguce!")
	}

	fmt.Println("Broj peerova: ", len(peerList))
	fmt.Println("\nkraj testnog ispisa za komunikaciju s trackerom\n----------------------------------")
	fmt.Println("")
	//Concurrency
	numPieces := len(t.Pieces) / 20

	peerPool := make(chan tracker.Peer, len(peerList)) // raspoloživi sudionici
	pieceJobs := make(chan int, numPieces)             // indeksi komada koje je potrebno preuzeti
	results := make(chan pieceResult, numPieces)       //preuzeti i provjereni komadi

	for _, p := range peerList {
		peerPool <- p
	}
	close(peerPool)

	for i := 0; i < numPieces; i++ {
		pieceJobs <- i
	}
	//close(pieceJobs) ????? ___________________________________________

	// TCP konekcija s peerom i handshake
	handshake, err := peer.BuildHandshake(t.InfoHash, peerID)
	if err != nil {
		fmt.Println("greska prilikom izrade handshakea")
	}

	var wg sync.WaitGroup

	numWorkers := min(10, len(peerList))
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(t, handshake, peerPool, pieceJobs, results, numPieces, &wg)
	}

	file, err := os.Create(t.Name)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	completed := 0
	for completed < numPieces {
		r := <-results
		_, err := file.WriteAt(r.data, int64(r.index)*int64(t.PieceLength))
		if err != nil {
			log.Fatal(err)
		}
		completed++
		/*if completed == numPieces {
			close(pieceJobs)
		}*/
		//fmt.Printf("Piece %d je preuzet i spremljen, preuzeto %d/%d\n", r.index, completed, numPieces)
		fmt.Printf("\rpreuzeto %d/%d", completed, numPieces)
	}

	//wg.Wait()
	//close(pieceJobs)
	fmt.Println("\nPreuzimanje zavrseno!")
}

/*func findWorkingConnection(peerPool chan tracker.Peer, handshake []byte) net.Conn {
peerLoop:
	for p := range peerPool {
		address := fmt.Sprintf("%s:%s", p.IP, strconv.Itoa(int(p.Port)))
		conn, err := peer.ConnectToPeer(handshake, address)
		if err != nil {
			continue
		}

		conn.Write(peer.BuildInterested())

		conn.SetReadDeadline(time.Now().Add(peer.UnchokeTimeout))
		for {
			msg, err := peer.ReadMessage(conn)
			if err != nil {
				conn.Close()
				continue peerLoop
			}

			if msg == nil {
				continue
			}

			if msg.ID == peer.MsgUnchoke {
				fmt.Printf("Spojeno na %s, Unchoke poruka primljena\n", address)
				return conn
			}
		}

	}
	return nil

}*/
