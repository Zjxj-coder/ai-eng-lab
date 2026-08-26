package imgproc

func ProcessImageGo(data []byte) []byte {
	res := make([]byte, len(data))
	for i, b := range data {
		res[i] = 255 - b
	}
	return res
}
