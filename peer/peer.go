package peer

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
	"torrentclient/torrent"
	"torrentclient/tracker"
)

const (
	handshakeTimeout = 5 * time.Second
	blockTimeout     = 20 * time.Second
	UnchokeTimeout   = 15 * time.Second
)

const (
	MsgChoke         byte = 0
	MsgUnchoke       byte = 1
	MsgInterested    byte = 2
	MsgNotInterested byte = 3
	MsgHave          byte = 4
	MsgBitfield      byte = 5
	MsgRequest       byte = 6
	MsgPiece         byte = 7
	MsgCancel        byte = 8
)

type Message struct {
	ID      byte
	Payload []byte
}

// izrada handshake poruke prema pravilima BitTorrent protokola
func BuildHandshake(infoHash [20]byte, peerID [20]byte) ([]byte, error) {

	handshake := make([]byte, 68)
	reserved := make([]byte, 8)
	handshake[0] = 19

	text := "BitTorrent protocol"
	copy(handshake[1:], []byte(text))
	copy(handshake[20:], reserved)
	copy(handshake[28:], infoHash[:])
	copy(handshake[48:], peerID[:])

	return handshake, nil
}

// spajanje na peer
func ConnectToPeer(handshake []byte, address string) (net.Conn, error) {

	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("greska prilikom otvaranja tcp veze %w", err)
	}
	conn.SetDeadline(time.Now().Add(handshakeTimeout))

	_, err = conn.Write(handshake)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("greska prilikom handshakea %w", err)
	}

	response := make([]byte, 68)

	_, err = io.ReadFull(conn, response)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("greska prilikom primanja handshakea %w", err)
	}

	if bytes.Equal(response[28:48], handshake[28:48]) {

		conn.SetDeadline(time.Time{})
		return conn, nil
	} else {
		fmt.Println("Neuspješan handshake")
		conn.Close()
		return nil, fmt.Errorf("Neuspješan handshake %w", err)
	}

}

// funkcija za čitanje odgovora peera
func ReadMessage(conn net.Conn) (*Message, error) {
	lengthBuff := make([]byte, 4)
	_, err := io.ReadFull(conn, lengthBuff)
	if err != nil {
		return nil, fmt.Errorf("greska prilikom citanja duljine poruke %w", err)
	}

	length := binary.BigEndian.Uint32(lengthBuff)

	if length == 0 {
		return nil, nil
	}

	messageBuff := make([]byte, length)
	_, err = io.ReadFull(conn, messageBuff)

	if err != nil {
		return nil, fmt.Errorf("greska prilikom citanja poruke %w", err)
	}

	message := Message{
		ID:      messageBuff[0],
		Payload: messageBuff[1:],
	}

	return &message, nil
}

func BuildRequest(index, begin, length int) []byte {
	buffer := make([]byte, 17)
	binary.BigEndian.PutUint32(buffer[0:4], 13)
	buffer[4] = MsgRequest
	// jer sad trazimo samo piece 0 pa ide 0
	binary.BigEndian.PutUint32(buffer[5:9], uint32(index))
	//begin offset za prvi blok nula za drugi 16384 (16 kb) itd.
	binary.BigEndian.PutUint32(buffer[9:13], uint32(begin))
	// length
	binary.BigEndian.PutUint32(buffer[13:17], uint32(length))

	return buffer
}

func DownloadPiece(t *torrent.TorrentInfo, conn net.Conn, index int, pieceLength int) ([]byte, error) {
	offset := 0
	pieceData := make([]byte, pieceLength)

	for offset < pieceLength {
		blockLen := min(16384, pieceLength-offset)
		buffer := BuildRequest(index, offset, blockLen)

		n, err := conn.Write(buffer)
		if err != nil {
			return nil, fmt.Errorf("greška prilikom slanja Build requesta %w", err)
		}
		if n != len(buffer) {
			return nil, fmt.Errorf("greška prilikom slanja Build requesta, broj poslanih bajtova nije jednak očekivanom broju %w", err)
		}

		for {
			conn.SetReadDeadline(time.Now().Add(blockTimeout))
			msg, err := ReadMessage(conn)
			if err != nil {
				return nil, fmt.Errorf("greška prilikom primanja poruke %w", err)
			}

			if msg == nil {
				continue
			}

			if msg.ID == MsgPiece {
				begin := binary.BigEndian.Uint32(msg.Payload[4:8])
				copy(pieceData[begin:], msg.Payload[8:])
				offset += blockLen
				break
			}
		}
	}

	hash := sha1.Sum(pieceData)
	expectedHash := t.PieceHashes()[index]

	if hash == expectedHash {
		return pieceData, nil
	} else {
		fmt.Println("Hash se ne podudara")
		return nil, errors.New("krivi hash")
	}
}

func BuildInterested() []byte {
	buffer := make([]byte, 5)
	binary.BigEndian.PutUint32(buffer, 1)
	buffer[4] = MsgInterested

	return buffer
}

func FindWorkingConnection(peerPool chan tracker.Peer, handshake []byte) net.Conn {
peerLoop:
	for p := range peerPool {
		address := fmt.Sprintf("%s:%s", p.IP, strconv.Itoa(int(p.Port)))
		conn, err := ConnectToPeer(handshake, address)
		if err != nil {
			continue
		}

		conn.Write(BuildInterested())

		conn.SetReadDeadline(time.Now().Add(UnchokeTimeout))
		for {
			msg, err := ReadMessage(conn)
			if err != nil {
				conn.Close()
				continue peerLoop
			}

			if msg == nil {
				continue
			}

			if msg.ID == MsgUnchoke {
				//fmt.Printf("Spojeno na %s, Unchoke poruka primljena\n", address)
				return conn
			}
		}

	}
	return nil

}
