package kvass

import (
	"bytes"
	"testing"
)

func TestCrypto(t *testing.T) {
	t.Parallel()
	p, err := NewSqlitePersistance(":memory:")
	if err != nil {
		t.Error(err)
	}
	text := []byte("Hello Crypto!")
	enc, err := p.Encrypt(text)
	if err != nil {
		t.Error(err)
	}
	dec, err := p.DecryptData(enc)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(text, dec) {
		t.Error("Payload changed!")
	}

}
