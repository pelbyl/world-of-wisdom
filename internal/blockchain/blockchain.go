package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type Block struct {
	Index        int
	Timestamp    time.Time
	Data         string
	PreviousHash string
	Hash         string
}

type Blockchain struct {
	blocks []Block
	mu     sync.RWMutex
}

func NewBlockchain() *Blockchain {
	bc := &Blockchain{
		blocks: make([]Block, 0),
	}
	// Create genesis block
	genesis := Block{
		Index:        0,
		Timestamp:    time.Now(),
		Data:         "Genesis Block",
		PreviousHash: "0",
	}
	genesis.Hash = bc.calculateHash(genesis)
	bc.blocks = append(bc.blocks, genesis)
	return bc
}

func (bc *Blockchain) calculateHash(block Block) string {
	record := fmt.Sprintf("%d%s%s%s", block.Index, block.Timestamp, block.Data, block.PreviousHash)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func (bc *Blockchain) AddBlock(data string) Block {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	prevBlock := bc.blocks[len(bc.blocks)-1]
	newBlock := Block{
		Index:        prevBlock.Index + 1,
		Timestamp:    time.Now(),
		Data:         data,
		PreviousHash: prevBlock.Hash,
	}
	newBlock.Hash = bc.calculateHash(newBlock)
	bc.blocks = append(bc.blocks, newBlock)
	return newBlock
}

func (bc *Blockchain) GetBlocks() []Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	// Return a copy to prevent external modification
	blocks := make([]Block, len(bc.blocks))
	copy(blocks, bc.blocks)
	return blocks
}

func (bc *Blockchain) GetLatestBlock() Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	return bc.blocks[len(bc.blocks)-1]
}

func (bc *Blockchain) GetBlockCount() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	return len(bc.blocks)
}