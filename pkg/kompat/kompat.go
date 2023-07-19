package kompat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/imdario/mergo"
	"github.com/mitchellh/go-homedir"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

var (
	DefaultFileName = "compatibility.yaml"
)

type Kompat struct {
	AppName       string          `yaml:"appName" json:"appName"`
	Displayed     int             `yaml:"displayed" json:"displayed"`
	Compatibility []Compatibility `yaml:"compatibility" json:"compatibility"`
}

type Compatibility struct {
	AppVersion    string `yaml:"appVersion" json:"appVersion"`
	MinK8sVersion string `yaml:"minK8sVersion" json:"minK8sVersion"`
	MaxK8sVersion string `yaml:"maxK8sVersion" json:"maxK8sVersion"`
}

func Parse(filePaths ...string) (Kompat, error) {
	var kompats []Kompat
	if len(filePaths) == 0 {
		filePaths = append(filePaths, ".")
	}
	for _, f := range filePaths {
		var contents []byte
		url, ok := toURL(f)
		if ok {
			if !strings.HasSuffix(url, ".yaml") {
				url = fmt.Sprintf("%s/blobs/main/%s", url, DefaultFileName)
			}
			resp, err := http.Get(url)
			if err != nil {
				return Kompat{}, err
			}
			defer resp.Body.Close()
			contents, err = io.ReadAll(resp.Body)
			if err != nil {
				return Kompat{}, err
			}
		} else {
			f, err := homedir.Expand(f)
			if err != nil {
				return Kompat{}, err
			}
			contents, err = os.ReadFile(f)
			if err != nil {
				return Kompat{}, err
			}
		}
		var kompat Kompat
		if err := yaml.Unmarshal(contents, &kompat); err != nil {
			return Kompat{}, err
		}
		kompats = append(kompats, kompat)
	}
	return Merge(kompats...), nil
}

func Merge(kompats ...Kompat) Kompat {
	var kompat Kompat
	for _, k := range kompats {
		mergo.Merge(&kompat, k)
	}
	return kompat
}

func (k Kompat) JSON() string {
	var buffer bytes.Buffer
	enc := json.NewEncoder(&buffer)
	enc.SetIndent("", "    ")
	if err := enc.Encode(k); err != nil {
		panic(err)
	}
	return buffer.String()
}

func (k Kompat) YAML() string {
	var buffer bytes.Buffer
	enc := yaml.NewEncoder(&buffer)
	if err := enc.Encode(k); err != nil {
		panic(err)
	}
	return buffer.String()
}

func (k Kompat) Markdown() string {
	out := bytes.Buffer{}
	table := tablewriter.NewWriter(&out)
	headers := []string{"K8s Versions"}
	data := []string{fmt.Sprintf("%s Versions", k.AppName)}
	for _, c := range k.Compatibility {
		headers = append(headers, fmt.Sprintf("%s - %s", c.MinK8sVersion, c.MaxK8sVersion))
		data = append(data, c.AppVersion)
	}
	table.SetHeader(headers)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.AppendBulk([][]string{data})
	table.Render()
	return out.String()
}

func toURL(str string) (string, bool) {
	isURL := false
	for _, t := range []string{".com", ".net", "http"} {
		if strings.Contains(str, t) {
			isURL = true
			break
		}
	}
	if !isURL {
		return "", false
	}
	if !strings.HasPrefix(str, "http") {
		str = fmt.Sprintf("%s%s", "https://", str)
	}
	url, err := url.Parse(str)
	if err != nil {
		return "", false
	}
	return url.String(), true
}
