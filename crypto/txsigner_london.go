package crypto

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/0xPolygon/polygon-edge/types"
)

// LondonSigner implements signer for EIP-1559
type LondonSigner struct {
	chainID        uint64
	isHomestead    bool
	fallbackSigner TxSigner
}

// NewLondonSigner returns a new LondonSigner object
func NewLondonSigner(chainID uint64, isHomestead bool, fallbackSigner TxSigner) *LondonSigner {
	return &LondonSigner{
		chainID:        chainID,
		isHomestead:    isHomestead,
		fallbackSigner: fallbackSigner,
	}
}

// type LondonSigner struct{ eip2930Signer }

// // NewLondonSigner returns a signer that accepts
// // - EIP-1559 dynamic fee transactions
// // - EIP-2930 access list transactions,
// // - EIP-155 replay protected transactions, and
// // - legacy Homestead transactions.
// func NewLondonSigner(chainId *big.Int) *LondonSigner {
// 	return LondonSigner{eip2930Signer{NewEIP155Signer(chainId)}}
// }

// Hash is a wrapper function that calls calcTxHash with the LondonSigner's fields
func (e *LondonSigner) Hash(tx *types.Transaction) types.Hash {
	return calcTxHash(tx, e.chainID)
}

// Sender returns the transaction sender
func (e *LondonSigner) Sender(tx *types.Transaction) (types.Address, error) {
	// Apply fallback signer for non-dynamic-fee-txs
	if tx.Type() != types.DynamicFeeTx {
		return e.fallbackSigner.Sender(tx)
	}

	v, r, s := tx.RawSignatureValues()
	sig, err := encodeSignature(r, s, v, e.isHomestead)
	if err != nil {
		return types.Address{}, err
	}

	pub, err := Ecrecover(e.Hash(tx).Bytes(), sig)
	if err != nil {
		return types.Address{}, err
	}

	buf := Keccak256(pub[1:])[12:]

	return types.BytesToAddress(buf), nil
}

// SignTx signs the transaction using the passed in private key
func (e *LondonSigner) SignTx(tx *types.Transaction, pk *ecdsa.PrivateKey) (*types.Transaction, error) {
	// Apply fallback signer for non-dynamic-fee-txs
	if tx.Type() != types.DynamicFeeTx {
		return e.fallbackSigner.SignTx(tx, pk)
	}

	tx = tx.Copy()

	h := e.Hash(tx)

	sig, err := Sign(pk, h[:])
	if err != nil {
		return nil, err
	}

	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:64])
	v := new(big.Int).SetBytes(e.calculateV(sig[64]))
	tx.SetSignatureValues(v, r, s)

	return tx, nil
}

// calculateV returns the V value for transaction signatures. Based on EIP155
func (e *LondonSigner) calculateV(parity byte) []byte {
	return big.NewInt(int64(parity)).Bytes()
}
