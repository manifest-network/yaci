package models

// Block represents a blockchain block.
type Block struct {
	ID   uint64
	Data []byte
}

// Transaction represents a blockchain transaction.
type Transaction struct {
	Hash string
	Data []byte
}
