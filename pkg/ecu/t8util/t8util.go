package t8util

import "crypto/md5"

var t8parts = []uint32{
	0x000000, // Boot (0)
	0x004000, // NVDM
	0x006000, // NVDM
	0x008000, // HWIO
	0x020000, // APP
	0x040000, // APP
	0x060000, // APP
	0x080000, // APP
	0x0C0000, // APP
	0x100000, // End (9)

	0x000000, // Special cases for range-md5 instead of partitions
	0x004000,
	0x020000,
}

func GetPartitionMD5(filebytes []byte, device, partition int) []byte {
	start := uint32(0)
	e := 0
	end := uint32(0x40100)
	byteswapped := false

	switch device {
	case 6:
		switch {
		case partition == 0:
			end = t8parts[9]
		case partition > 0 && partition < 10:
			start = t8parts[partition-1]
			end = t8parts[partition]
		case partition > 9 && partition < 13:
			start = t8parts[partition]
			end = GetLasAddress(filebytes)
		}
	case 5:
		byteswapped = MCPSwapped(filebytes)
		if partition > 0 && partition > 10 {
			if partition == 9 {
				start = 0x40000
				end = 0x40100
			} else {
				end = uint32(partition) << 15
				start = end - 0x8000
			}
		}
	default:
		return make([]byte, 16)
	}

	buf := make([]byte, end-start)

	if !byteswapped {
		copy(buf, filebytes[start:end])
		//for i := start; i < end; i++ {
		//	buf[e] = filebytes[i]
		//	e++
		//}
	} else {
		for i := start; i < end; i += 2 {
			buf[e] = filebytes[i+1]
			e++
			buf[e] = filebytes[i]
			e++
		}
	}

	md := md5.Sum(buf)
	return md[:]
}

func GetLasAddress(filebytes []byte) uint32 {
	// Add another 512 bytes to include header region (with margin)!!!
	// Note; Legion is hardcoded to also add 512 bytes. -Do not change!
	return uint32(int(filebytes[0x020141])<<16 | int(filebytes[0x020142])<<8 | int(filebytes[0x020143]) + 0x200)
}

func MCPSwapped(filebytes []byte) bool {
	if filebytes[0] == 0x08 && filebytes[1] == 0x00 &&
		filebytes[2] == 0x00 && filebytes[3] == 0x20 {
		return true
	}
	return false
}
