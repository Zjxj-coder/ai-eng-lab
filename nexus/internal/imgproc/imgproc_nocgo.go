//go:build !cgo

package imgproc

const HasCgo = false

func ProcessImageCgo(data []byte) []byte {
	panic("cgo not enabled, ProcessImageCgo should not be called")
}
