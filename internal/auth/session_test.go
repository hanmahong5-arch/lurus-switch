package auth

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// These tests cover the AES-256-GCM envelope that guards auth.enc on disk.
// A silent corruption here locks the user out of their session with no error
// signal, so the encrypt/decrypt round-trip, tamper detection, key stability,
// nonce uniqueness, and truncated-input handling all get explicit coverage.

// testKey returns a deterministic 32-byte key so the crypto tests don't depend
// on the host's hostname/username (which deriveEncryptionKey reads).
func testKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i)
	}
	return k
}

func TestEncryptDecryptAESGCM_RoundTrip(t *testing.T) {
	key := testKey()
	cases := [][]byte{
		[]byte(""),
		[]byte("hello"),
		[]byte(`{"access_token":"abc","refresh_token":"def"}`),
		bytes.Repeat([]byte("x"), 4096),
	}
	for _, plaintext := range cases {
		ct, err := encryptAESGCM(key, plaintext)
		if err != nil {
			t.Fatalf("encrypt(%d bytes): %v", len(plaintext), err)
		}
		got, err := decryptAESGCM(key, ct)
		if err != nil {
			t.Fatalf("decrypt(%d bytes): %v", len(plaintext), err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Errorf("round-trip mismatch: got %q, want %q", got, plaintext)
		}
	}
}

func TestDecryptAESGCM_TamperedCiphertextFails(t *testing.T) {
	key := testKey()
	ct, err := encryptAESGCM(key, []byte("sensitive token material"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Flip one bit in the ciphertext body (past the 12-byte nonce) — the GCM
	// MAC must reject it rather than return garbage plaintext or panic.
	tampered := make([]byte, len(ct))
	copy(tampered, ct)
	tampered[len(tampered)-1] ^= 0x01

	got, err := decryptAESGCM(key, tampered)
	if err == nil {
		t.Fatalf("tampered ciphertext decrypted without error: %q", got)
	}
}

func TestDecryptAESGCM_WrongKeyFails(t *testing.T) {
	ct, err := encryptAESGCM(testKey(), []byte("payload"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	wrong := make([]byte, 32)
	wrong[0] = 0xFF // differs from testKey()[0]==0x00
	if _, err := decryptAESGCM(wrong, ct); err == nil {
		t.Fatal("decrypt with wrong key should fail the MAC, got nil error")
	}
}

func TestDecryptAESGCM_TruncatedAndEmptyFailCleanly(t *testing.T) {
	key := testKey()
	// Empty input is shorter than the nonce → "ciphertext too short", no panic.
	if _, err := decryptAESGCM(key, nil); err == nil {
		t.Error("empty ciphertext should error")
	}
	if _, err := decryptAESGCM(key, []byte{}); err == nil {
		t.Error("zero-length ciphertext should error")
	}
	// A buffer shorter than the GCM nonce (12 bytes) must be rejected, not
	// indexed out of bounds.
	if _, err := decryptAESGCM(key, []byte{1, 2, 3}); err == nil {
		t.Error("sub-nonce-length ciphertext should error")
	}
	// Nonce-length-only (no actual ciphertext+tag) is still too short for the
	// GCM tag and must error rather than panic.
	if _, err := decryptAESGCM(key, make([]byte, 12)); err == nil {
		t.Error("nonce-only ciphertext (no tag) should error")
	}
}

func TestEncryptAESGCM_NonceIsUniquePerCall(t *testing.T) {
	key := testKey()
	plaintext := []byte("same plaintext both times")
	a, err := encryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt a: %v", err)
	}
	b, err := encryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt b: %v", err)
	}
	// The first 12 bytes are the prepended nonce — a reused nonce under GCM is
	// catastrophic, so they must differ across calls.
	const nonceSize = 12
	if bytes.Equal(a[:nonceSize], b[:nonceSize]) {
		t.Error("nonce reused across two encryptions — GCM security broken")
	}
	// Same plaintext + same key + different nonce → different ciphertext.
	if bytes.Equal(a, b) {
		t.Error("two encryptions of the same plaintext produced identical output")
	}
}

func TestEncryptAESGCM_RejectsBadKeyLength(t *testing.T) {
	// AES requires a 16/24/32-byte key; a short key must surface an error, not
	// a panic, so a corrupt derivation is observable.
	if _, err := encryptAESGCM([]byte("too-short"), []byte("data")); err == nil {
		t.Error("expected error for invalid key length")
	}
}

func TestDeriveEncryptionKey_StableAndCorrectLength(t *testing.T) {
	k1, err := deriveEncryptionKey()
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	k2, err := deriveEncryptionKey()
	if err != nil {
		t.Fatalf("derive again: %v", err)
	}
	if len(k1) != 32 {
		t.Errorf("key length = %d, want 32 (AES-256)", len(k1))
	}
	if !bytes.Equal(k1, k2) {
		t.Error("deriveEncryptionKey is not deterministic for the same host/user")
	}
}

func TestEncryptDecrypt_RandomKeyRoundTrip(t *testing.T) {
	// Belt-and-suspenders: a freshly random 32-byte key still round-trips,
	// confirming the helpers don't depend on testKey()'s structure.
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand key: %v", err)
	}
	plaintext := []byte("round trip with a random key")
	ct, err := encryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := decryptAESGCM(key, ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("mismatch: got %q, want %q", got, plaintext)
	}
}
