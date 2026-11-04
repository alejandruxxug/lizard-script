package evaluator

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"hash/fnv"
)

// Digests: text.digest(s, algorithm) here, and fs.digest(path, algorithm)
// in the Terrarium, which takes the same algorithms through Digest.

// DigestAlgorithms are the names a digest takes, the default first.
var DigestAlgorithms = []string{"sha256", "fnv"}

// Digest is the hash of data with the named algorithm, as lowercase hex:
// "sha256" (64 digits, for telling contents apart when it matters) or
// "fnv" (FNV-1a of 64 bits, 16 digits, fast but easy to collide on
// purpose). ok is false for any other name.
func Digest(data []byte, algorithm string) (digest string, ok bool) {
	switch algorithm {
	case "sha256":
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:]), true
	case "fnv":
		h := fnv.New64a()
		h.Write(data)
		return hex.EncodeToString(h.Sum(nil)), true
	}
	return "", false
}

// textDigest is text.digest(s, algorithm = "sha256"): the hash of the
// text's bytes (UTF-8), as hex. The same text always gives the same
// digest, on every machine.
func textDigest(args ...object.Object) object.Object {
	if err := checkCount(args, 1, 2); err != nil {
		return err
	}
	s, err := stringArg(args, 0)
	if err != nil {
		return err
	}
	algorithm := DigestAlgorithms[0]
	if len(args) == 2 {
		if algorithm, err = stringArg(args, 1); err != nil {
			return err
		}
	}
	digest, ok := Digest([]byte(s), algorithm)
	if !ok {
		return UnknownDigest(algorithm)
	}
	return &object.String{Value: digest}
}

// UnknownDigest is the error of an algorithm Digest doesn't know.
func UnknownDigest(algorithm string) *object.Error {
	return argError("unknown algorithm %q, the algorithms are \"sha256\" and \"fnv\"", algorithm)
}
