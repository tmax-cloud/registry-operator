package dockerhub

import (
	"fmt"
	"testing"

	"github.com/bmizerany/assert"
)

func TestParseName(t *testing.T) {
	type suite struct {
		fullName      string
		expNamespace  string
		expRepository string
		expError      string
	}
	testCases := []suite{
		{
			fullName:      "tomcat",
			expNamespace:  "library",
			expRepository: "tomcat",
		},
		{
			fullName:      "tomcat:8.5",
			expNamespace:  "library",
			expRepository: "tomcat",
		},
		{
			fullName:      "library/alpine",
			expNamespace:  "library",
			expRepository: "alpine",
		},
		{
			fullName:      "test/alpine",
			expNamespace:  "test",
			expRepository: "alpine",
		},
		{
			fullName:      "test/al..pine",
			expNamespace:  "",
			expRepository: "",
			expError:      "invalid reference format",
		},
		{
			fullName:      "test/",
			expNamespace:  "",
			expRepository: "",
			expError:      "invalid reference format",
		},
		{
			fullName:      "test/test/test",
			expNamespace:  "",
			expRepository: "",
			expError:      "name must be namespace/repository",
		},
	}

	for i, testCase := range testCases {
		fmt.Println(i, "test")
		namespace, repository, err := ParseName(testCase.fullName)
		if testCase.expError != "" {
			if err == nil {
				fmt.Printf("expected: %s\n", testCase.expError)
				t.Fatal()
			}
			fmt.Println(err.Error())
			assert.Equal(t, testCase.expError, err.Error())
		}
		if testCase.expNamespace != "" && testCase.expRepository != "" {
			assert.Equal(t, testCase.expNamespace, namespace)
			assert.Equal(t, testCase.expRepository, repository)
		}
	}

}
