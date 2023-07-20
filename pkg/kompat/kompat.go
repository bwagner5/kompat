/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package kompat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/mitchellh/go-homedir"
	"github.com/olekukonko/tablewriter"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

var (
	DefaultFileName = "compatibility.yaml"
)

type KompatList []Kompat

type Kompat struct {
	AppName       string          `yaml:"appName" json:"appName"`
	Compatibility []Compatibility `yaml:"compatibility" json:"compatibility"`
}

type Compatibility struct {
	AppVersion    string `yaml:"appVersion" json:"appVersion"`
	MinK8sVersion string `yaml:"minK8sVersion" json:"minK8sVersion"`
	MaxK8sVersion string `yaml:"maxK8sVersion" json:"maxK8sVersion"`
}

type Options struct {
	LastN   int
	Version string
}

func Parse(filePaths ...string) (KompatList, error) {
	var kompats []Kompat
	if len(filePaths) == 0 {
		filePaths = append(filePaths, DefaultFileName)
	}
	for _, f := range filePaths {
		var contents []byte
		var err error
		url, ok := toURL(f)
		if ok {
			contents, err = readFromURL(url)
			if err != nil {
				return nil, err
			}
		} else {
			contents, err = readFromFile(f)
			if err != nil {
				return nil, err
			}
		}
		decoder := yaml.NewDecoder(bytes.NewBuffer(contents))
		for {
			var kompat Kompat
			err := decoder.Decode(&kompat)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, err
			}
			if err := kompat.Validate(); err != nil {
				return nil, err
			}
			kompats = append(kompats, kompat)
		}
	}
	return kompats, nil
}

func (k Kompat) Validate() error {
	for _, c := range k.Compatibility {
		appVersion := strings.ReplaceAll(c.AppVersion, ".x", "")
		minK8sVersion := strings.ReplaceAll(c.MinK8sVersion, ".x", "")
		maxK8sVersion := strings.ReplaceAll(c.MaxK8sVersion, ".x", "")
		if _, err := semver.NewVersion(appVersion); err != nil {
			return fmt.Errorf("unable to parse compatibility for \"%s\": appVersion \"%s\" is invalid: %w", k.AppName, c.AppVersion, err)
		}
		if _, err := semver.NewVersion(minK8sVersion); err != nil {
			return fmt.Errorf("unable to parse compatibility for \"%s\": minK8sVersion \"%s\" is invalid: %w", k.AppName, c.MinK8sVersion, err)
		}
		if _, err := semver.NewVersion(maxK8sVersion); err != nil {
			return fmt.Errorf("unable to parse compatibility for \"%s\": maxK8sVersion \"%s\" is invalid: %w", k.AppName, c.MaxK8sVersion, err)
		}
	}
	return nil
}

func (k Kompat) JSON() string {
	return KompatList{k}.JSON()
}

func (k KompatList) JSON() string {
	var buffer bytes.Buffer
	enc := json.NewEncoder(&buffer)
	enc.SetIndent("", "    ")
	if err := enc.Encode(k); err != nil {
		panic(err)
	}
	return buffer.String()
}

func (k Kompat) YAML() string {
	return KompatList{k}.YAML()
}

func (k KompatList) YAML() string {
	var buffer bytes.Buffer
	enc := yaml.NewEncoder(&buffer)
	if err := enc.Encode(k); err != nil {
		panic(err)
	}
	return buffer.String()
}

func (k Kompat) Markdown(opts ...Options) string {
	// options := mergeOptions(opts...)
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

func (k KompatList) Markdown(opts ...Options) string {
	options := mergeOptions(opts...)
	if len(k) == 1 {
		return k[0].Markdown()
	}
	out := bytes.Buffer{}
	table := tablewriter.NewWriter(&out)
	headers := []string{"K8s Versions"}
	var data [][]string
	// Get all k8s versions for the first row
	k8sVersions := k.k8sVersions()
	if options.Version != "" {
		version, ok := lo.Find(k8sVersions, func(version string) bool { return version == options.Version })
		if !ok {
			return ""
		}
		headers = append(headers, version)
	} else if options.LastN != 0 {
		headers = append(headers, k8sVersions[len(k8sVersions)-options.LastN:]...)
	} else {
		headers = append(headers, k8sVersions...)
	}

	// Fill in App version rows
	for i, app := range k {
		data = append(data, []string{})
		k8sVersionToAppVersions := app.expand()
		for j, k8sVersion := range headers {
			// skip the first column since it's the text header
			if j == 0 {
				data[i] = append(data[i], fmt.Sprintf("%s Versions", app.AppName))
				continue
			}
			allAppVersions := lo.Uniq(lo.Flatten(lo.Values(k8sVersionToAppVersions)))
			data[i] = append(data[i], semverRange(k8sVersionToAppVersions[k8sVersion], allAppVersions...))
		}
	}
	table.SetHeader(headers)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.AppendBulk(data)
	table.Render()
	return out.String()
}

func mergeOptions(opts ...Options) Options {
	if len(opts) == 0 {
		return Options{}
	}
	return opts[0]
}

func (k KompatList) k8sVersions() []string {
	var k8sVersions []string
	for _, app := range k {
		k8sVersions = append(k8sVersions, lo.Keys(app.expand())...)
	}
	k8sVersions = lo.Uniq(k8sVersions)
	sort.Slice(k8sVersions, func(i, j int) bool {
		return lo.Must(strconv.Atoi(strings.ReplaceAll(k8sVersions[i], ".", ""))) <
			lo.Must(strconv.Atoi(strings.ReplaceAll(k8sVersions[j], ".", "")))
	})
	return k8sVersions
}

// expand returns a map of K8s version to app version, expanding out ranges to single versions
func (k Kompat) expand() map[string][]string {
	k8sToApp := map[string][]string{}
	for _, e := range k.Compatibility {
		for _, kv := range k8sVersions(e.MinK8sVersion, e.MaxK8sVersion) {
			k8sToApp[kv] = append(k8sToApp[kv], e.AppVersion)
		}
	}
	return k8sToApp
}

// Helper functions

func k8sVersions(min string, max string) []string {
	var versions []string
	major := strings.Split(min, ".")[0]
	minMinor := lo.Must(strconv.Atoi(strings.Split(min, ".")[1]))
	maxMinor := lo.Must(strconv.Atoi(strings.Split(max, ".")[1]))
	for i := minMinor; i <= maxMinor; i++ {
		versions = append(versions, fmt.Sprintf("%s.%d", major, i))
	}
	return versions
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

func readFromURL(url string) ([]byte, error) {
	if !strings.HasSuffix(url, ".yaml") {
		if strings.Contains(url, "github.com") {
			url = fmt.Sprintf("%s/main/%s", url, DefaultFileName)
			url = strings.Replace(url, "github.com", "raw.githubusercontent.com", 1)
		}
	}
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

func readFromFile(file string) ([]byte, error) {
	var err error
	file, err = homedir.Expand(file)
	if err != nil {
		return nil, err
	}
	contents, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

// semverRange will sort the versions and output in a pretty range string in the format of "1.21 - 1.27"
// if allSemvers is passed, then semverRange will check if the max version is equal to the max in allSemvers
// if it is then you can get prettier strings like "1.21+"
func semverRange(semvers []string, allSemvers ...string) string {
	if len(semvers) == 0 {
		return ""
	}
	if len(semvers) == 1 {
		return semvers[0]
	}
	sortSemvers(semvers)
	if len(allSemvers) != 0 {
		allSems := allSemvers
		sortSemvers(allSems)
		if allSems[len(allSems)-1] == semvers[len(semvers)-1] {
			return fmt.Sprintf("%s+", semvers[0])
		}
	}
	return fmt.Sprintf("%s - %s", semvers[0], semvers[len(semvers)-1])
}

func sortSemvers(semvers []string) {
	sort.Slice(semvers, func(i, j int) bool {
		return semver.MustParse(strings.ReplaceAll(semvers[i], ".x", "")).LessThan(semver.MustParse(strings.ReplaceAll(semvers[j], ".x", "")))
	})
}
