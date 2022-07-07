package util

func IsEthAddress(address string) bool {
	if len(address) != 42 {
		return false
	}
	if address[0] != '0' || address[1] != 'x' {
		return false
	}
	for _, c := range address[2:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}
