package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/scrypt"
)

var (
	// Scrypt parameters (tunable; tests may override)
	ScryptN = 1 << 15 // 32768
	ScryptR = 8
	ScryptP = 1
	KeyLen  = 32
)

const (
	magic    = "GPBK" // file magic
	version  = 0x01
	saltLen  = 16
	nonceLen = chacha20poly1305.NonceSizeX
)

// ErrBadFormat indicates the encrypted file is invalid or corrupt.
var ErrBadFormat = errors.New("bad backup format")

// createTarGzip collects the provided path (directory) into gzipped tar bytes.
func createTarGzip(root string) ([]byte, error) {
	buf := &bytes.Buffer{}
	gw := gzip.NewWriter(buf)
	tr := tar.NewWriter(gw)

	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// skip root
		if rel == "." {
			return nil
		}
		// ensure forward slashes for tar header name (cross-platform compatibility)
		rel = filepath.ToSlash(rel)

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tr.WriteHeader(hdr); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tr, f); err != nil {
				return err
			}
		}
		return nil
	}

	if err := filepath.Walk(root, walk); err != nil {
		return nil, err
	}
	// ensure writers are closed in order
	if err := tr.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// deriveKey derives a 32-byte key from passphrase and salt using scrypt.
func deriveKey(pass []byte, salt []byte) ([]byte, error) {
	key, err := scrypt.Key(pass, salt, ScryptN, ScryptR, ScryptP, KeyLen)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// zeroBytes overwrites the slice with zeros.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Backup writes an encrypted backup file for baseDir to outPath using passphrase.
func Backup(baseDir, outPath string, passphrase []byte) error {
	// create archive
	payload, err := createTarGzip(baseDir)
	if err != nil {
		return err
	}

	// generate salt & derive key
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return err
	}
	defer zeroBytes(key)

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return err
	}

	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	ciphertext := aead.Seal(nil, nonce, payload, nil)

	// write file: magic(4), version(1), salt(16), nonce(24), timestamp(8), len(ciphertext 8), ciphertext
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write([]byte(magic)); err != nil {
		return err
	}
	if _, err := f.Write([]byte{version}); err != nil {
		return err
	}
	if _, err := f.Write(salt); err != nil {
		return err
	}
	if _, err := f.Write(nonce); err != nil {
		return err
	}
	// timestamp
	ts := time.Now().Unix()
	var tsbuf [8]byte
	binary.BigEndian.PutUint64(tsbuf[:], uint64(ts))
	if _, err := f.Write(tsbuf[:]); err != nil {
		return err
	}
	var lenbuf [8]byte
	binary.BigEndian.PutUint64(lenbuf[:], uint64(len(ciphertext)))
	if _, err := f.Write(lenbuf[:]); err != nil {
		return err
	}
	if _, err := f.Write(ciphertext); err != nil {
		return err
	}
	return nil
}

// Restore decrypts inPath to destDir using passphrase.
func Restore(inPath, destDir string, passphrase []byte) error {
	b, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}
	if len(b) < 4+1+saltLen+nonceLen+8+8 {
		return ErrBadFormat
	}
	off := 0
	if string(b[:4]) != magic {
		return ErrBadFormat
	}
	off += 4
	if b[off] != version {
		return fmt.Errorf("unsupported version: %d", b[off])
	}
	off++
	salt := b[off : off+saltLen]
	off += saltLen
	nonce := b[off : off+nonceLen]
	off += nonceLen
	// timestamp skip
	off += 8
	cLen := binary.BigEndian.Uint64(b[off : off+8])
	off += 8
	if int(cLen) != len(b)-off {
		return ErrBadFormat
	}
	ciphertext := b[off:]

	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return err
	}
	defer zeroBytes(key)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return err
	}
	payload, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	// write payload (gzipped tar) into destDir
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return err
	}
	gr, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		default:
			// skip other types
		}
	}
	return nil
}
