package sia

import (
	"crypto/ecdsa"
	"math/big"
)

const (
	HashSize      = 32
	PublicKeySize = 32
	SignatureSize = 32
	SegmentSize   = 32

	TargetSecondsPerBlock = 300
	DifficultyWindow      = 5000
)

var MaxAdjustmentUp = big.NewRat(10005, 10000)
var MaxAdjustmentDown = big.NewRat(9995, 10000)

type (
	Hash      [HashSize]byte
	PublicKey ecdsa.PublicKey

	Timestamp   int64
	BlockHeight uint32
	Currency    uint64

	BlockID       Hash
	OutputID      Hash // An output id points to a specific output.
	ContractID    Hash
	TransactionID Hash
	CoinAddress   Hash // An address points to spend conditions.
	Difficulty    Hash
)

type Signature struct {
	R, S *big.Int
}

type Block struct {
	Version      uint16
	ParentBlock  BlockID
	Timestamp    Timestamp
	Nonce        uint32 // may or may not be needed
	MinerAddress CoinAddress
	MerkleRoot   Hash
	Transactions []Transaction
}

type Transaction struct {
	Version       uint16
	ArbitraryData []byte
	MinerFee      Currency
	Inputs        []Input
	Outputs       []Output
	FileContracts []FileContract
	StorageProofs []StorageProof
	Signatures    []TransactionSignature
}

type Input struct {
	OutputID        OutputID // the source of coins for the input
	SpendConditions SpendConditions
}

type Output struct {
	Value     Currency // how many coins are in the output
	SpendHash CoinAddress
}

type SpendConditions struct {
	TimeLock      BlockHeight
	NumSignatures uint8
	PublicKeys    []PublicKey
}

type TransactionSignature struct {
	InputID        OutputID // the OutputID of the Input that this signature is addressing. Using the index has also been considered.
	PublicKeyIndex uint8
	TimeLock       BlockHeight
	CoveredFields  CoveredFields
	Signature      Signature
}

type CoveredFields struct {
	Version         bool
	ArbitraryData   bool
	MinerFee        bool
	Inputs, Outputs []uint8 // each element indicates an index which is signed.
	Contracts       []uint8
	FileProofs      []uint8
}

// Not enough flexibility in payments?  With the Start and End times, the only
// problem is if the client wishes for the rewards or penalties to scale as
// more are submitted or missed, and if they want things to scale harder in the
// case of consecutive misses.
type FileContract struct {
	ContractFund       Currency
	FileMerkleRoot     Hash
	FileSize           uint64 // probably in bytes, which means the last element in the merkle tree may not be exactly 64 bytes.
	Start, End         BlockHeight
	ChallengeFrequency uint32 // size of window, one window at a time
	Tolerance          uint32 // number of missed proofs before triggering unsuccessful termination
	ValidProofPayout   Currency
	ValidProofAddress  CoinAddress
	MissedProofPayout  Currency
	MissedProofAddress CoinAddress
	SuccessAddress     CoinAddress
	FailureAddress     CoinAddress
}

type StorageProof struct {
	ContractID ContractID
	Segment    [SegmentSize]byte
	HashSet    []*Hash
}

// MarshalSia implements the Marshaler interface for Signatures.
func (s *Signature) MarshalSia() []byte {
	if s.R == nil || s.S == nil {
		return []byte{0, 0}
	}
	// pretend Signature is a tuple of []bytes
	// this lets us use Marshal instead of doing manual length-prefixing
	return Marshal(struct{ R, S []byte }{s.R.Bytes(), s.S.Bytes()})
}

// MarshalSia implements the Unmarshaler interface for Signatures.
func (s *Signature) UnmarshalSia(b []byte) int {
	// inverse of the struct trick used in Signature.MarshalSia
	str := struct{ R, S []byte }{}
	if Unmarshal(b, &str) != nil {
		return 0
	}
	s.R = new(big.Int).SetBytes(str.R)
	s.S = new(big.Int).SetBytes(str.S)
	return len(str.R) + len(str.S) + 2
}

// MarshalSia implements the Marshaler interface for PublicKeys.
func (pk *PublicKey) MarshalSia() []byte {
	if pk.X == nil || pk.Y == nil {
		return []byte{0, 0}
	}
	// see Signature.MarshalSia
	return Marshal(struct{ X, Y []byte }{pk.X.Bytes(), pk.Y.Bytes()})
}

// MarshalSia implements the Unmarshaler interface for PublicKeys.
func (pk *PublicKey) UnmarshalSia(b []byte) int {
	// see Signature.UnmarshalSia
	str := struct{ X, Y []byte }{}
	if Unmarshal(b, &str) != nil {
		return 0
	}
	pk.X = new(big.Int).SetBytes(str.X)
	pk.Y = new(big.Int).SetBytes(str.Y)
	return len(str.X) + len(str.Y) + 2
}

// MerkleRoot calculates the Merkle root hash of a SpendConditions object,
// using the timelock, number of signatures, and the signatures themselves as leaves.
func (sc *SpendConditions) MerkleRoot() Hash {
	tlHash := HashBytes(Marshal(sc.TimeLock))
	nsHash := HashBytes(Marshal(sc.NumSignatures))
	pkHashes := make([]Hash, len(sc.PublicKeys))
	for i := range sc.PublicKeys {
		pkHashes[i] = HashBytes(Marshal(sc.PublicKeys[i]))
	}
	leaves := append([]Hash{tlHash, nsHash}, pkHashes...)
	return MerkleRoot(leaves)
}