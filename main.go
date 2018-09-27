package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/hashicorp/hcl"
	"github.com/mbndr/logo"
	. "gopkg.in/src-d/go-git.v4/plumbing/transport"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"github.com/olekukonko/tablewriter"
	"reflect"
	"sync"
	"time"
)

var log = logo.NewSimpleLogger(os.Stderr, logo.INFO, "versionGetter ", true)
var StatsInfo = NewStats()
var files []string
var config Config
var wg sync.WaitGroup

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type ListFileData struct  {
	Data []FileData
}
type Module struct {
	Name    string
	Type    string
	Source  string
	Version string
	Path    string
	gitInfo GitInfo
}

type FileData struct {
	Name    string
	Path    string
	Modules []Module
}

func (s *Stats) addFile() {
	s.Files++
}

func (s *Stats) addModule() {
	s.Modules++
}

func NewStats() *Stats {
	return &Stats{StartTime: time.Now()}

}

func (s *Stats) Report() string {
	return fmt.Sprintf("Number of Files: %v. Number of Modules %v", s.Files, s.Modules)
}

type FileContents struct {
	Item []map[string][]struct {
		Source  string `hcl:"source" json:"source"`
		Version string `hcl:"version" json:"version"`
	} `hcl:"module" json:"module"`
}

func parsePayload(paylaod []byte) (FileContents, error) {
	var elem = FileContents{}
	err := hcl.Unmarshal(paylaod, &elem)
	if err != nil {
		return FileContents{}, err
	}
	return elem, nil

}

type Config struct {
	SrcPath string
	Verbose bool
	Stats   bool
	ExtraSkip arrayFlags
	Output  string
}

func parseSrc(source string) *Endpoint {
	endpoint, err := NewEndpoint(source)
	if err != nil {
		panic(err)
	}
	return endpoint
}

type GitInfo struct {
	Owner      string
	Repository string
	Ref        string
	SubPath    string
}

func SplitGitUrl(str string) GitInfo {
	regStr := `(?P<owner>[\w\d]+)\/(?P<reponame>[\w\d]+.git)\??(ref=(?P<ref>((\d+\.)?(\d+\.)?(\*|\d+))|[\w\d]+))?(\/\/(?P<path>[\w\d]+))?`
	regOperator, err := regexp.Compile(regStr)
	if err != nil {
		panic(err)
	}

	match := regOperator.FindStringSubmatch(str)
	result := make(map[string]string)
	for i, name := range regOperator.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	gitInfo := GitInfo{Owner: result["owner"],
		Repository: result["reponame"],
		Ref:        result["ref"],
		SubPath:    result["path"]}
	log.Debug(gitInfo)
	return gitInfo
}

func processFile(path string) (FileData, error) {
	fileContents, _ := ioutil.ReadFile(path)
	fileData := FileData{Name: filepath.Base(path),
		Path: path}

	log.Debugf(fmt.Sprintf("Processing file %s", path))
	StatsInfo.addFile()
	element, err := parsePayload(fileContents)
	if err != nil {
		return FileData{}, err
	}
	for _, item := range element.Item {
		if len(item) >= 1 {
			for k, v := range item {
				StatsInfo.addModule()
				thisMod := Module{Name: k, Source: v[0].Source, Version: v[0].Version}
				endpoint := parseSrc(thisMod.Source)
				thisMod.Type = endpoint.Protocol
				switch endpoint.Protocol {
				case "ssh":
					thisMod.gitInfo = SplitGitUrl(endpoint.Path)
					thisMod.Version = thisMod.gitInfo.Ref
					thisMod.Path = thisMod.gitInfo.SubPath
					thisMod.Source = thisMod.gitInfo.Repository


				case "file":
					if thisMod.Version != "" {
						thisMod.Type = "registry"
					}
					thisMod.Path = endpoint.Path
				}
				fileData.Modules = append(fileData.Modules, thisMod)
			}
		}
	}
	return fileData, nil

}

type Stats struct {
	StartTime time.Time
	EndTime time.Time
	Files   int
	Modules int
	Duration time.Duration
}

func (s *Stats) Stop() {
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
}

func listFiles(path string, f os.FileInfo, err error) error {
	isToSkip, _ := in_array(f.Name(), []string{".git", ".terraform"})
	log.Debugf("Processing %s", f.Name())
	if f.IsDir() && isToSkip {
		log.Debugf("Skipping dir %s", f.Name())
		return filepath.SkipDir
	}
	if filepath.Ext(path) == ".tf" {
		files = append(files, path)
	}
	return nil
}

func in_array(val interface{}, array interface{}) (exists bool, index int) {
	exists = false
	index = -1

	switch reflect.TypeOf(array).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(array)

		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(val, s.Index(i).Interface()) == true {
				index = i
				exists = true
				return
			}
		}
	}
	return
}

func main() {

	config.Output = "text"
	// Add basic folders to exlude
	config.ExtraSkip.Set(".git")
	config.ExtraSkip.Set(".terraform")

	flag.StringVar(&config.SrcPath, "source-path", "", "The path to the source directory to scan")
	flag.BoolVar(&config.Verbose, "verbose", false, "be verbose")
	flag.BoolVar(&config.Stats, "stats", false, "Show stats")
	flag.Var(&config.ExtraSkip, "extra-skip", "Extra skip dirs")
	flag.StringVar(&config.Output, "output", "text", "output format (text|json)")
	flag.Parse()
	if config.Verbose {
		log.SetLevel(logo.DEBUG)
	}
	var Data ListFileData
	err := filepath.Walk(config.SrcPath, listFiles)
	//c := make(chan error)
	//go func() { c <- filepath.Walk(config.SrcPath, listFiles) }()
	//err := <-c // Walk done, check the error
	if err != nil {
		panic(err)
	}
	if config.SrcPath == "" {
		flag.Usage()
		os.Exit(1)
	} else {
		search := fmt.Sprintf("%s", config.SrcPath)
		log.Infof("search in %s", search)
		//paths, err := filepath.Glob(search)
		if err != nil {
			panic(err)
		}

		for _, file := range files {
			fileData, err := processFile(file)
			if err != nil {
				panic(err)
			}
			if fileData.Modules != nil {
				Data.Data = append(Data.Data, fileData)
			}
		}
	}
	StatsInfo.Stop()
	// Cleanup
	switch config.Output {
	case "text":
		Data.TableWriter()
	case "json":
		DisplayElement(Data)

	}
	//DisplayElement(Data)
	if config.Stats {
		log.Infof(StatsInfo.Report())
		log.Infof("Took %s", StatsInfo.Duration)
	}
}
func (ld *ListFileData) TableWriter() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(ld.Header())

	for _, v := range ld.PrepareTabulare() {
		table.Append(v)
	}
	table.Render() // Send output

}

func (ld *ListFileData) Header() []string {
	return []string{"File", "Module", "Type", "Source", "Version"}
}


func (ld *ListFileData) PrepareTabulare() [][]string {
	var output [][]string
	for _, item := range ld.Data {
		for _, v := range item.Modules {
			add := []string{item.Path, v.Name, v.Type, v.Source, v.Version}
			output = append(output, add)
		}
	}
	return output
}
func DisplayElement(data ListFileData) {
	// Process Elems
	jsonPayload, err := json.MarshalIndent(data.Data, "", "    ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(jsonPayload))
}
