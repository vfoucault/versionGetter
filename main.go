package main

import (
	"github.com/hashicorp/hcl"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"io/ioutil"
	"github.com/mbndr/logo"
	"encoding/json"
	. "gopkg.in/src-d/go-git.v4/plumbing/transport"
	"regexp"
)
var log = logo.NewSimpleLogger(os.Stderr, logo.DEBUG, "prefix ", true)

type Module struct {
	Name string
	Type string
	Source string
	Version string
	Path string
	gitInfo GitInfo
}

type FileData struct {
	Name string
	Path string
	Modules []Module
}

type FileContents struct {
	Item []map[string][]struct{
		Source string `hcl:"source" json:"source"`
		Version string `hcl:"version" json:"version"`
	} `hcl:"module" json:"module"`
}

func parsePayload(paylaod []byte) (FileContents, error){
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
}

func parseSrc(source string) *Endpoint {
	endpoint, err := NewEndpoint(source)
	if err != nil {
		panic(err)
	}
	return endpoint
}

type GitInfo struct {
	Owner	string
	Repository string
	Ref string
	SubPath string
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
		Ref: result["ref"],
		SubPath: result["path"]}
	log.Debug(gitInfo)
	return gitInfo
}


func processFile(path string) (FileData, error) {
	fileContents, _ := ioutil.ReadFile(path)
	fileData := FileData{Name: filepath.Base(path),
						 Path: path}

	log.Infof(fmt.Sprintf("Processing file %s", path))
	element, err:= parsePayload(fileContents)
	if err != nil {
		return FileData{}, err
	}
	for _, item := range element.Item {
		if len(item) >= 1 {
			for k, v := range item {
				thisMod := Module{Name: k, Source: v[0].Source, Version: v[0].Version}
				endpoint := parseSrc(thisMod.Source)
				thisMod.Type = endpoint.Protocol
				switch endpoint.Protocol {
				case "ssh": thisMod.gitInfo = SplitGitUrl(endpoint.Path)
							thisMod.Version = thisMod.gitInfo.Ref
							thisMod.Path = thisMod.gitInfo.SubPath
				case "file": thisMod.Path = endpoint.Path


				}
				fileData.Modules = append(fileData.Modules, thisMod)
			}
		}
	}
	return fileData, nil

}



func main() {
	var config Config

	flag.StringVar(&config.SrcPath,  "source-path", "", "The path to the source directory to scan")
	flag.BoolVar(&config.Verbose,  "verbose", false, "be verbose")
	flag.Parse()

	var Data []FileData
	if config.SrcPath == "" {
		flag.Usage()
		os.Exit(1)
	} else {
		search := fmt.Sprintf("%s/**/*.tf", config.SrcPath)
		log.Infof("search in %s", search)
		paths, err := filepath.Glob(search)
		if err != nil {
			panic(err)
		}

		for _, file := range paths {
			fileData, err := processFile(file)
			if err != nil {
				panic(err)
			}
			if fileData.Modules != nil {
				Data = append(Data, fileData)
			}
		}
	}
	// Cleanup

	DisplayElement(Data)


}

func DisplayElement(elements []FileData) {
	// Process Elems
	json, err := json.MarshalIndent(elements, "", "    ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(json))
}