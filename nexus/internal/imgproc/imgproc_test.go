package imgproc

import (
	"bytes"
	"testing"
)

func TestImgproc_Compare(t *testing.T) {
	if !HasCgo {
		t.Skip("Skipping cgo vs go comparison: cgo is not enabled")
	}

	input := []byte{0, 10, 127, 200, 255}
	
	resGo := ProcessImageGo(input)
	resCgo := ProcessImageCgo(input)
	
	if !bytes.Equal(resGo, resCgo) {
		t.Fatalf("results mismatch: Go %v, Cgo %v", resGo, resCgo)
	}
}
