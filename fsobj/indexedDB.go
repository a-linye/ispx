package fsobj

import (
	"context"
	"io/fs"
	"log"
	"strings"
	"time"
	"syscall/js"
    "io"
	"fmt"
    "bytes"
	"encoding/base64"
    "io/ioutil"
	"encoding/json"
)

type IndexDBFs struct {
	ctx    context.Context
	root   string
	cur    time.Time
	dirEntry string
}

type webEntry struct {
	name  string
	isDir bool
}

type JSONEntry struct {
    Name     string      `json:"name"`
    IsDir    bool        `json:"isDir"`
    Children []JSONEntry `json:"children,omitempty"`
}

type IndexDBDir struct {
	ctx    context.Context
	assert string
	cache  map[string]string
}

func NewIndexDBDir(assert string) (*IndexDBDir, error) {
	return &IndexDBDir{
		ctx:    context.Background(),
		assert: assert,
		cache:  make(map[string]string),
	}, nil
}

func (d *IndexDBDir) Open(file string) (io.ReadCloser, error) {
	
	// 创建一个用于异步 JavaScript 调用的 channel
	done := make(chan struct{})
	var data []byte
	var err error
    log.Println("Open:", file)
	// 调用 JavaScript 函数
    filepath := d.assert + "/" + file
	js.Global().Call("readDataFromIndexedDB", filepath).Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// 解码 Base64 字符串
		base64Data := args[0].String()
		data, err = base64.StdEncoding.DecodeString(base64Data)

		done <- struct{}{}
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err = fmt.Errorf("error reading file: %s", args[0].String())
		done <- struct{}{}
		return nil
	}))

	// 等待异步操作完成
	<-done

	// 将数据包装成io.ReadCloser接口
	reader := bytes.NewReader(data)
	closer := ioutil.NopCloser(reader)

	return closer, err
}

func (d *IndexDBDir) Close() error {
	return nil
}

func (e *webEntry) Name() string {
	// 获取文件名
	return e.name 
}
func (e *webEntry) IsDir() bool {
	// 判断是否为目录
	return e.isDir 
}

func (e *webEntry) Type() fs.FileMode {
	if e.IsDir() {
		return fs.ModeDir
	}
	return 0
}
func (e *webEntry) Info() (fs.FileInfo, error) {
	return nil, nil
}



func NewIndexDBFs(root string) (*IndexDBFs, error) {
	return &IndexDBFs{ctx: context.Background(), root: root, cur: time.Now()}, nil
}


func (f *IndexDBFs) ReadDir(filename string) ([]fs.DirEntry, error) {

	global := js.Global().Get("top")
    textarea := global.Get("document").Call("getElementById", "dirStructure")
    jsValue := textarea.Get("value").String()

    // 解析 JSON 数据
    var jsonEntries []JSONEntry
    err := json.Unmarshal([]byte(jsValue), &jsonEntries)
    if err != nil {
        return nil, err
    }

    // 初始化 entries 列表
    var entries []webEntry
    processEntries(jsonEntries, "", &entries)

    // 准备返回的列表
    var list []fs.DirEntry
    for i := range entries {
        list = append(list, &entries[i])
    }

    if Verbose {
        log.Println("ReadDir", filename, list)
    }

    return list, nil
}

func processEntries(jsonEntries []JSONEntry, currentPath string, entries *[]webEntry) {
    for _, entry := range jsonEntries {
        fullPath := entry.Name
        if currentPath != "" {
            fullPath = currentPath + "/" + entry.Name
        }

        *entries = append(*entries, webEntry{name: fullPath, isDir: entry.IsDir})

        if entry.IsDir && len(entry.Children) > 0 {
            processEntries(entry.Children, fullPath, entries)
        }
    }
}

func (f *IndexDBFs) ReadFile(filename string) ([]byte, error) {
	// 创建一个用于异步 JavaScript 调用的 channel
	done := make(chan struct{})
	var data []byte
	var err error

    log.Println("ReadFile ing....")

	// 调用 JavaScript 函数
	js.Global().Call("readDataFromIndexedDB", filename).Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// 解码 Base64 字符串
		base64Data := args[0].String()
		data, err = base64.StdEncoding.DecodeString(base64Data)

		done <- struct{}{}
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err = fmt.Errorf("error reading file: %s", args[0].String())
		done <- struct{}{}
		return nil
	}))

	// 等待异步操作完成
	<-done

	return data, err
}

func (f *IndexDBFs) Join(elem ...string) string {
	return strings.Join(elem, "/")
}
