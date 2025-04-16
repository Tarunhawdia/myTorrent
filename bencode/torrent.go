package bencode

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceLength int
	Pieces      [][20]byte
	Name        string
	Length      int
}