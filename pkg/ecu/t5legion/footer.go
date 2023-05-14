package t5legion

import (
	"context"
	"log"
	"strings"
)

//var ecuFooter []byte

func (t *Client) GetECUFooter(ctx context.Context) ([]byte, error) {
	//if !t.bootloaded {
	//	if err := t.UploadBootLoader(ctx); err != nil {
	//		return err
	//	}
	//}
	if len(t.ecuFooter) > 0 {
		return t.ecuFooter, nil
	}

	var address uint32 = 0x80000 - 0x80

	footer, _, err := t.readDataByLocalIdentifier(ctx, 0x06, int(address), 0x80)
	if err != nil {
		return nil, err
	}

	t.ecuFooter = footer
	return footer, nil
}

func GetIdentifierFromFooter(footer []byte, identifier byte) string {
	var result strings.Builder
	offset := len(footer) - 0x05 //  avoid the stored checksum
	for offset > 0 {
		length := int(footer[offset])
		offset--
		search := footer[offset]
		offset--
		if identifier == search {
			for i := 0; i < length; i++ {
				result.WriteByte(footer[offset])
				offset--
			}
			return result.String()
		}
		offset -= length
	}
	log.Printf("error getting identifier 0x%X", identifier)
	return ""
}
