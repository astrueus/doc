package model

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"git.itopcms.com/astrueus/doc/pkg/filetil"
	"git.itopcms.com/astrueus/doc/pkg/ziptil"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/russross/blackfriday/v2"
)

// 导入项目
func (book *Book) ImportBook(zipPath string, lang string) error {
	if !filetil.FileExists(zipPath) {
		return errors.New("文件不存在 => " + zipPath)
	}

	w := md5.New()
	io.WriteString(w, zipPath) //将str写入到w中
	io.WriteString(w, time.Now().String())
	io.WriteString(w, book.BookName)
	md5str := fmt.Sprintf("%x", w.Sum(nil)) //w.Sum(nil)将w的hash转成[]byte格式

	tempPath := filepath.Join(os.TempDir(), md5str)

	if err := os.MkdirAll(tempPath, 0766); err != nil {
		logs.Error("创建导入目录出错 => ", err)
	}
	//如果加压缩失败
	if err := ziptil.Unzip(zipPath, tempPath); err != nil {
		return err
	}
	//当导入结束后，删除临时文件
	//defer os.RemoveAll(tempPath)

	for {
		//如果当前目录下只有一个目录，则重置根目录
		if entries, err := os.ReadDir(tempPath); err == nil && len(entries) == 1 {
			dir := entries[0]
			if dir.IsDir() && dir.Name() != "." && dir.Name() != ".." {
				tempPath = filepath.Join(tempPath, dir.Name())
				break
			}

		} else {
			break
		}
	}

	tempPath = strings.Replace(tempPath, "\\", "/", -1)

	docMap := make(map[string]int, 0)

	o := orm.NewOrm()

	o.Insert(book)
	relationship := NewRelationship()
	relationship.BookId = book.BookId
	relationship.RoleId = 0
	relationship.MemberId = book.MemberId
	relationship.Insert()

	err := filepath.Walk(tempPath, func(path string, info os.FileInfo, err error) error {
		path = strings.Replace(path, "\\", "/", -1)
		if path == tempPath {
			return nil
		}
		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			//如果是Markdown文件
			if strings.EqualFold(ext, ".md") || strings.EqualFold(ext, ".markdown") {
				logs.Info("正在处理 =>", path, info.Name())
				doc := NewDocument()
				doc.BookId = book.BookId
				doc.MemberId = book.MemberId
				docIdentify := strings.Replace(strings.TrimPrefix(path, tempPath+"/"), "/", "-", -1)

				if ok, err := regexp.MatchString(`[a-z]+[a-zA-Z0-9_.\-]*$`, docIdentify); !ok || err != nil {
					docIdentify = "import-" + docIdentify
				}

				doc.Identify = docIdentify
				//匹配图片，如果图片语法是在代码块中，这里同样会处理
				re := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)
				markdown, err := filetil.ReadFileAndIgnoreUTF8BOM(path)
				if err != nil {
					return err
				}

				//处理图片 / 链接（逻辑见 book_import_images.go）
				doc.Markdown = re.ReplaceAllStringFunc(string(markdown), func(image string) string {
					return rewriteImportMarkdownImage(image, path, tempPath, re)
				})

				linkRegexp := regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
				doc.Markdown = linkRegexp.ReplaceAllStringFunc(doc.Markdown, func(link string) string {
					return rewriteImportMarkdownLink(link, path, tempPath, book.Identify, linkRegexp)
				})

				//codeRe := regexp.MustCompile("```\\w+")

				//doc.Markdown = codeRe.ReplaceAllStringFunc(doc.Markdown, func(s string) string {
				//	//logs.Info(s)
				//	return strings.Replace(s,"```","``` ",-1)
				//})

				doc.Content = string(blackfriday.Run([]byte(doc.Markdown)))

				doc.Version = time.Now().Unix()

				//解析文档名称，默认使用第一个h标签为标题
				docName := strings.TrimSuffix(info.Name(), ext)

				for _, line := range strings.Split(doc.Markdown, "\n") {
					if strings.HasPrefix(line, "#") {
						docName = strings.TrimLeft(line, "#")
						break
					}
				}

				doc.DocumentName = strings.TrimSpace(docName)

				parentId := 0

				parentIdentify := strings.Replace(strings.Trim(strings.TrimSuffix(strings.TrimPrefix(path, tempPath), info.Name()), "/"), "/", "-", -1)

				if parentIdentify != "" {

					if ok, err := regexp.MatchString(`[a-z]+[a-zA-Z0-9_.\-]*$`, parentIdentify); !ok || err != nil {
						parentIdentify = "import-" + parentIdentify
					}
					if id, ok := docMap[parentIdentify]; ok {
						parentId = id
					}
				}
				if strings.EqualFold(info.Name(), "README.md") {
					logs.Info(path, "|", info.Name(), "|", parentIdentify, "|", parentId)
				}
				isInsert := false
				//如果当前文件是README.md，则将内容更新到父级
				if strings.EqualFold(info.Name(), "README.md") && parentId != 0 {

					doc.DocumentId = parentId
					//logs.Info(path,"|",parentId)
				} else {
					//logs.Info(path,"|",parentIdentify)
					doc.ParentId = parentId
					isInsert = true
				}
				if err := doc.InsertOrUpdate("document_name", "markdown", "content"); err != nil {
					logs.Error(doc.DocumentId, err)
				}
				if isInsert {
					docMap[docIdentify] = doc.DocumentId
				}
			}
		} else {
			//如果当前目录下存在Markdown文件，则需要创建此节点
			if filetil.HasFileOfExt(path, []string{".md", ".markdown"}) {
				logs.Info("正在处理 =>", path, info.Name())
				identify := strings.Replace(strings.Trim(strings.TrimPrefix(path, tempPath), "/"), "/", "-", -1)
				if ok, err := regexp.MatchString(`[a-z]+[a-zA-Z0-9_.\-]*$`, identify); !ok || err != nil {
					identify = "import-" + identify
				}

				parentDoc := NewDocument()

				parentDoc.MemberId = book.MemberId
				parentDoc.BookId = book.BookId
				parentDoc.Identify = identify
				parentDoc.Version = time.Now().Unix()
				parentDoc.DocumentName = "空白文档"

				parentId := 0

				parentIdentify := strings.TrimSuffix(identify, "-"+info.Name())

				if id, ok := docMap[parentIdentify]; ok {
					parentId = id
				}

				parentDoc.ParentId = parentId

				if err := parentDoc.InsertOrUpdate(); err != nil {
					logs.Error(err)
				}

				docMap[identify] = parentDoc.DocumentId
				//logs.Info(path,"|",parentDoc.DocumentId,"|",identify,"|",info.Name(),"|",parentIdentify)
			}
		}

		return nil
	})

	if err != nil {
		logs.Error("导入项目异常 => ", err)
		book.Description = "【项目导入存在错误：" + err.Error() + "】"
	}
	logs.Info("项目导入完毕 => ", book.BookName)
	book.ReleaseContent(book.BookId, lang)
	return err
}
