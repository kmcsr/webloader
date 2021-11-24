
package webopener

import (
	os "os"
	io "io"
	strings "strings"
	regexp "regexp"
	hex "encoding/hex"
	adler32 "hash/adler32"
)

var empty_re = regexp.MustCompile(`\s+`)

func zipString(str string)(string){
	if len(str) == 0 {
		return str
	}
	return empty_re.ReplaceAllString(str, " ")
}

func strToBool(v string)(bool){
	switch strings.ToUpper(v) {
	case "TRUE", "T", "OK", "YES", "1":
		return true
	}
	return false
}

func calculateFileHash(path string)(_ string, err error){
	var (
		fd *os.File
	)
	fd, err = os.Open(path)
	if err != nil { return }
	defer fd.Close()
	h := adler32.New()
	_, err = io.Copy(h, fd)
	if err != nil { return }
	return hex.EncodeToString(h.Sum(make([]byte, 0, 4))), nil
}

func calculateHash(bts []byte)(string){
	h := adler32.New()
	h.Write(bts)
	return hex.EncodeToString(h.Sum(make([]byte, 0, 4)))
}

func zipCodeJs(str string)(string){
	return strings.TrimSpace(str)
}

func zipCodeCss(str string)(string){
	return zipString(str)
}


