# BitTorrent klijent

Pojednostavljeni BitTorrent klijent napisan u programskom jeziku Go, u cijelosti
uporabom standardne biblioteke i bez vanjskih ovisnosti. Izrađen kao završni rad.

Implementira vlastiti bencode raščlanjivač, izračun info hasha, komunikaciju s
poslužiteljem za praćenje, peer wire protokol te konkurentno preuzimanje s
provjerom integriteta svakog komada algoritmom SHA-1.

# Zahtjevi

- Go 1.26 ili novija verzija
- 
# Pokretanje

Klijent se pokreće preko terminala naredbom "go run ." i putanjom do .torrent datoteke kao jedinim argumentom

Primjer:
```bash
go run . datoteka.torrent
```

## Podržano

- Torrenti s jednom datotekom
- Poslužitelji za praćenje preko HTTP-a i HTTPS-a

## Nije podržano

- Torrenti s više datoteka
- UDP poslužitelji za praćenje, DHT i magnet poveznice
- Slanje podataka drugim sudionicima (klijent radi samo kao preuzimatelj)
- Nastavak prekinutog preuzimanja


