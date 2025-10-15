package iamlivecore

import (
	"regexp"
	"strings"
)

// uniqueStringList is a small helper set+slice structure for ARN template resolution.
type uniqueStringList struct {
	list []string
	set  map[string]bool
}

func newUniqueStringList() *uniqueStringList {
	return &uniqueStringList{set: map[string]bool{}}
}

func (s *uniqueStringList) add(newArn string) {
	if _, ok := s.set[newArn]; !ok {
		s.list = append(s.list, newArn)
		s.set[newArn] = true
	}
}

func (s *uniqueStringList) addParam(arns []string, paramVarName, param string) {
	for _, arn := range arns {
		newArn := regexp.MustCompile(`\$\{`+strings.ReplaceAll(strings.ReplaceAll(paramVarName, "[", "\\["), "]", "\\]")+`\}`).ReplaceAllString(arn, param)
		s.add(newArn)
	}
}
