package uppaal

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

func escapeForXML(text string) string {
	buf := new(bytes.Buffer)
	err := xml.EscapeText(buf, []byte(text))
	if err != nil {
		panic(fmt.Errorf("could not escape xml text: %v", err))
	}
	return buf.String()
}
