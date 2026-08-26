//go:build cgo

package imgproc

/*
#include <stdlib.h>
void process_image_c(unsigned char* data, int length) {
    for (int i=0; i<length; i++) {
        data[i] = 255 - data[i];
    }
}
*/
import "C"
import "unsafe"

const HasCgo = true

func ProcessImageCgo(data []byte) []byte {
	if len(data) == 0 {
		return []byte{}
	}
	res := make([]byte, len(data))
	copy(res, data)
	C.process_image_c((*C.uchar)(unsafe.Pointer(&res[0])), C.int(len(res)))
	return res
}
