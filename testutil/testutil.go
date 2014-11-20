package testutil

import (
	"bufio"
	"net"
	"regexp"
)

func ReadIO(conn net.Conn) string {
	res, _ := bufio.NewReader(conn).ReadString('\n')
	return res
}

func MatchRegex(regex string, target string) bool {
	re, err := regexp.Compile(regex)
	if err != nil {
		panic(err)
	}
	return re.MatchString(target)
}
