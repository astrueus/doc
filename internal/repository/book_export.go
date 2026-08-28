package repository

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"net/http"
	"regexp"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/converter"
	"git.itopcms.com/astrueus/doc/internal/dto"
	"git.itopcms.com/astrueus/doc/internal/model"
	"git.itopcms.com/astrueus/doc/pkg/cryptil"
	"git.itopcms.com/astrueus/doc/pkg/filetil"
	"git.itopcms.com/astrueus/doc/pkg/gopool"
	"git.itopcms.com/astrueus/doc/pkg/requests"
	"git.itopcms.com/astrueus/doc/pkg/ziptil"
	"github.com/PuerkitoBio/goquery"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/russross/blackfriday/v2"
)

var (
	exportLimitWorkerChannel = gopool.NewChannelPool(config.GetExportLimitNum(), config.GetExportQueueLimitNum())
)

func BackgroundConvert(sessionId string, bookResult *dto.BookResult) error {

	if err := converter.CheckConvertCommand(); err != nil {
		logs.Error("检查转换程序失败 -> ", err)
		return err
	}
	err := exportLimitWorkerChannel.LoadOrStore(bookResult.Identify, func() {
		ConvertBook(sessionId, bookResult)
	})

	if err != nil {

		logs.Error("将导出任务加入任务队列失败 -> ", err)
		return err
	}
	exportLimitWorkerChannel.Start()
	return nil
}

// ConvertBook 导出 PDF、word 等格式（原 BookResult.Converter；dto 不能挂方法）。
func ConvertBook(sessionId string, m *dto.BookResult) (dto.ConvertBookResult, error) {

	convertBookResult := dto.ConvertBookResult{}

	outputPath := filepath.Join(config.GetExportOutputPath(), strconv.Itoa(m.BookId))
	viewPath := config.WorkingDir("web", "views")

	pdfpath := filepath.Join(outputPath, "book.pdf")
	epubpath := filepath.Join(outputPath, "book.epub")
	mobipath := filepath.Join(outputPath, "book.mobi")
	docxpath := filepath.Join(outputPath, "book.docx")

	//先将转换的文件储存到临时目录
	tempOutputPath := filepath.Join(os.TempDir(), sessionId, m.Identify, "source") //filepath.Abs(filepath.Join("cache", sessionId))

	sourceDir := strings.TrimSuffix(tempOutputPath, "source")
	if filetil.FileExists(sourceDir) {
		if err := os.RemoveAll(sourceDir); err != nil {
			logs.Error("删除临时目录失败 ->", sourceDir, err)
		}
	}

	if err := os.MkdirAll(outputPath, 0766); err != nil {
		logs.Error("创建目录失败 -> ", outputPath, err)
	}
	if err := os.MkdirAll(tempOutputPath, 0766); err != nil {
		logs.Error("创建目录失败 -> ", tempOutputPath, err)
	}
	os.MkdirAll(filepath.Join(tempOutputPath, "Images"), 0755)

	//defer os.RemoveAll(strings.TrimSuffix(tempOutputPath,"source"))

	if filetil.FileExists(pdfpath) && filetil.FileExists(epubpath) && filetil.FileExists(mobipath) && filetil.FileExists(docxpath) {
		convertBookResult.EpubPath = epubpath
		convertBookResult.MobiPath = mobipath
		convertBookResult.PDFPath = pdfpath
		convertBookResult.WordPath = docxpath
		return convertBookResult, nil
	}

	docs, err := NewDocumentRepo(nil).FindListByBookID(context.Background(), m.BookId)
	if err != nil {
		return convertBookResult, err
	}

	tocList := make([]converter.Toc, 0)

	for _, item := range docs {
		if item.ParentId == 0 {
			toc := converter.Toc{
				Id:    item.DocumentId,
				Link:  strconv.Itoa(item.DocumentId) + ".html",
				Pid:   item.ParentId,
				Title: item.DocumentName,
			}

			tocList = append(tocList, toc)
		}
	}
	for _, item := range docs {
		if item.ParentId != 0 {
			toc := converter.Toc{
				Id:    item.DocumentId,
				Link:  strconv.Itoa(item.DocumentId) + ".html",
				Pid:   item.ParentId,
				Title: item.DocumentName,
			}
			tocList = append(tocList, toc)
		}
	}

	ebookConfig := converter.Config{
		Charset:      "utf-8",
		Cover:        m.Cover,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		Description:  string(blackfriday.Run([]byte(m.Description))),
		Footer:       "<p style='color:#8E8E8E;font-size:12px;'>本文档使用 <a href='https://www.iminho.me' style='text-decoration:none;color:#1abc9c;font-weight:bold;'>MinDoc</a> 构建 <span style='float:right'>- _PAGENUM_ -</span></p>",
		Header:       "<p style='color:#8E8E8E;font-size:12px;'>_SECTION_</p>",
		Identifier:   "",
		Language:     "zh-CN",
		Creator:      m.CreateName,
		Publisher:    m.Publisher,
		Contributor:  m.Publisher,
		Title:        m.BookName,
		Format:       []string{"epub", "mobi", "pdf", "docx"},
		FontSize:     "14",
		PaperSize:    "a4",
		MarginLeft:   "72",
		MarginRight:  "72",
		MarginTop:    "72",
		MarginBottom: "72",
		Toc:          tocList,
		More:         []string{},
	}
	if m.Publisher != "" {
		ebookConfig.Footer = "<p style='color:#8E8E8E;font-size:12px;'>本文档由 <span style='text-decoration:none;color:#1abc9c;font-weight:bold;'>" + m.Publisher + "</span> 生成<span style='float:right'>- _PAGENUM_ -</span></p>"
	}
	if m.RealName != "" {
		ebookConfig.Creator = m.RealName
	}

	if tempOutputPath, err = filepath.Abs(tempOutputPath); err != nil {
		logs.Error("导出目录配置错误：" + err.Error())
		return convertBookResult, err
	}

	for _, item := range docs {
		name := strconv.Itoa(item.DocumentId)
		fpath := filepath.Join(tempOutputPath, name+".html")

		f, err := os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, 0755)
		if err != nil {
			return convertBookResult, err
		}
		var buf bytes.Buffer

		if err := web.ExecuteViewPathTemplate(&buf, "document/export.tpl", viewPath, map[string]any{"Model": m, "Lists": item, "BaseUrl": config.BaseUrl}); err != nil {
			return convertBookResult, err
		}
		html := buf.String()

		if err != nil {

			f.Close()
			return convertBookResult, err
		}

		bufio := bytes.NewReader(buf.Bytes())

		doc, err := goquery.NewDocumentFromReader(bufio)
		doc.Find("img").Each(func(i int, contentSelection *goquery.Selection) {
			if src, ok := contentSelection.Attr("src"); ok {
				//var encodeString string
				dstSrcString := "Images/" + filepath.Base(src)

				//如果是本地路径则直接读取文件内容
				if strings.HasPrefix(src, "/") {
					spath := filepath.Join(config.WorkingDirectory, src)
					if filetil.CopyFile(spath, filepath.Join(tempOutputPath, dstSrcString)); err != nil {
						logs.Error("复制图片失败 -> ", err, src)
						return
					}

				} else {
					client := &http.Client{}
					if req, err := http.NewRequest("GET", src, nil); err == nil {
						req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.99 Safari/537.36")
						req.Header.Set("Referer", src)
						//10秒连接超时时间
						client.Timeout = time.Second * 100

						if resp, err := client.Do(req); err == nil {

							defer resp.Body.Close()

							if body, err := io.ReadAll(resp.Body); err == nil {
								//encodeString = base64.StdEncoding.EncodeToString(body)
								if err := os.WriteFile(filepath.Join(tempOutputPath, dstSrcString), body, 0755); err != nil {
									logs.Error("下载图片失败 -> ", err, src)
									return
								}
							} else {
								logs.Error("下载图片失败 -> ", err, src)
								return
							}
						} else {
							logs.Error("下载图片失败 -> ", err, src)
							return
						}
					}
				}
				contentSelection.SetAttr("src", dstSrcString)
			}
		})
		//移除文档底部的更新信息
		if selection := doc.Find("div.wiki-bottom").First(); selection.Size() > 0 {
			selection.Remove()
		}

		html, err = doc.Html()
		if err != nil {
			f.Close()
			return convertBookResult, err
		}
		f.WriteString(html)
		f.Close()
	}

	if err := filetil.CopyFile(filepath.Join(config.WorkingDirectory, "web", "static", "css", "kancloud.css"), filepath.Join(tempOutputPath, "styles", "css", "kancloud.css")); err != nil {
		logs.Error("复制CSS样式出错 -> static/css/kancloud.css", err)
	}
	if err := filetil.CopyFile(filepath.Join(config.WorkingDirectory, "web", "static", "css", "export.css"), filepath.Join(tempOutputPath, "styles", "css", "export.css")); err != nil {
		logs.Error("复制CSS样式出错 -> static/css/export.css", err)
	}
	if err := filetil.CopyFile(filepath.Join(config.WorkingDirectory, "web", "static", "editor.md", "css", "editormd.preview.css"), filepath.Join(tempOutputPath, "styles", "editor.md", "css", "editormd.preview.css")); err != nil {
		logs.Error("复制CSS样式出错 -> static/editor.md/css/editormd.preview.css", err)
	}

	if err := filetil.CopyFile(filepath.Join(config.WorkingDirectory, "web", "static", "css", "markdown.preview.css"), filepath.Join(tempOutputPath, "styles", "css", "markdown.preview.css")); err != nil {
		logs.Error("复制CSS样式出错 -> static/css/markdown.preview.css", err)
	}
	if err := filetil.CopyFile(filepath.Join(config.WorkingDirectory, "web", "static", "editor.md", "lib", "highlight", "styles", "github.css"), filepath.Join(tempOutputPath, "styles", "css", "github.css")); err != nil {
		logs.Error("复制CSS样式出错 -> static/editor.md/lib/highlight/styles/github.css", err)
	}

	if err := filetil.CopyDir(filepath.Join(config.WorkingDirectory, "web", "static", "font-awesome"), filepath.Join(tempOutputPath, "styles", "font-awesome")); err != nil {
		logs.Error("复制CSS样式出错 -> static/font-awesome", err)
	}

	eBookConverter := &converter.Converter{
		BasePath:   tempOutputPath,
		OutputPath: filepath.Join(strings.TrimSuffix(tempOutputPath, "source"), "output"),
		Config:     ebookConfig,
		Debug:      true,
		ProcessNum: config.GetExportProcessNum(),
	}

	os.MkdirAll(eBookConverter.OutputPath, 0766)

	if err := eBookConverter.Convert(); err != nil {
		logs.Error("转换文件错误：" + m.BookName + " -> " + err.Error())
		return convertBookResult, err
	}
	logs.Info("文档转换完成：" + m.BookName)

	if err := filetil.CopyFile(filepath.Join(eBookConverter.OutputPath, "output", "book.mobi"), mobipath); err != nil {
		logs.Error("复制文档失败 -> ", filepath.Join(eBookConverter.OutputPath, "output", "book.mobi"), err)
	}
	if err := filetil.CopyFile(filepath.Join(eBookConverter.OutputPath, "output", "book.pdf"), pdfpath); err != nil {
		logs.Error("复制文档失败 -> ", filepath.Join(eBookConverter.OutputPath, "output", "book.pdf"), err)
	}
	if err := filetil.CopyFile(filepath.Join(eBookConverter.OutputPath, "output", "book.epub"), epubpath); err != nil {
		logs.Error("复制文档失败 -> ", filepath.Join(eBookConverter.OutputPath, "output", "book.epub"), err)
	}
	if err := filetil.CopyFile(filepath.Join(eBookConverter.OutputPath, "output", "book.docx"), docxpath); err != nil {
		logs.Error("复制文档失败 -> ", filepath.Join(eBookConverter.OutputPath, "output", "book.docx"), err)
	}

	convertBookResult.MobiPath = mobipath
	convertBookResult.PDFPath = pdfpath
	convertBookResult.EpubPath = epubpath
	convertBookResult.WordPath = docxpath

	return convertBookResult, nil
}

// ExportBookMarkdown 导出 Markdown 原始文件（原 BookResult.ExportMarkdown）。
func ExportBookMarkdown(sessionId string, m *dto.BookResult) (string, error) {
	outputPath := filepath.Join(config.WorkingDirectory, "uploads", "books", strconv.Itoa(m.BookId), "book.zip")

	os.MkdirAll(filepath.Dir(outputPath), 0644)

	tempOutputPath := filepath.Join(os.TempDir(), sessionId, "markdown")

	defer os.RemoveAll(tempOutputPath)

	bookUrl := config.URLFor("DocumentController.Index", ":key", m.Identify) + "/"

	err := exportMarkdown(tempOutputPath, 0, m.BookId, tempOutputPath, bookUrl)

	if err != nil {
		return "", err
	}

	if err := ziptil.Compress(outputPath, tempOutputPath); err != nil {
		logs.Error("导出Markdown失败->", err)
		return "", err
	}
	return outputPath, nil
}

// 递归导出Markdown文档
func exportMarkdown(p string, parentId int, bookId int, baseDir string, bookUrl string) error {
	o := orm.NewOrm()

	var docs []*model.Document

	_, err := o.QueryTable(model.NewDocument().TableNameWithPrefix()).Filter("book_id", bookId).Filter("parent_id", parentId).All(&docs)

	if err != nil {
		logs.Error("导出Markdown失败->", err)
		return err
	}
	for _, doc := range docs {
		//获取当前文档的子文档数量，如果数量不为0，则将当前文档命名为READMD.md并设置成目录。
		subDocCount, err := o.QueryTable(model.NewDocument().TableNameWithPrefix()).Filter("parent_id", doc.DocumentId).Count()

		if err != nil {
			logs.Error("导出Markdown失败->", err)
			return err
		}

		var docPath string

		if subDocCount > 0 {
			if doc.Identify != "" {
				docPath = filepath.Join(p, doc.Identify, "README.md")
			} else {
				docPath = filepath.Join(p, strconv.Itoa(doc.DocumentId), "README.md")
			}
		} else {
			if doc.Identify != "" {
				if strings.HasSuffix(doc.Identify, ".md") || strings.HasSuffix(doc.Identify, ".markdown") {
					docPath = filepath.Join(p, doc.Identify)
				} else {
					docPath = filepath.Join(p, doc.Identify+".md")
				}
			} else {
				docPath = filepath.Join(p, strings.TrimSpace(doc.DocumentName)+".md")
			}
		}
		dirPath := filepath.Dir(docPath)

		os.MkdirAll(dirPath, 0766)
		markdown := doc.Markdown
		//如果当前文档不为空
		if strings.TrimSpace(doc.Markdown) != "" {
			re := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)

			//处理文档中图片
			markdown = re.ReplaceAllStringFunc(doc.Markdown, func(image string) string {
				images := re.FindAllSubmatch([]byte(image), -1)
				if len(images) <= 0 || len(images[0]) < 3 {
					return image
				}
				originalImageUrl := string(images[0][2])
				imageUrl := strings.Replace(string(originalImageUrl), "\\", "/", -1)

				//如果是本地路径，则需要将图片复制到项目目录
				if strings.HasPrefix(imageUrl, "http://") || strings.HasPrefix(imageUrl, "https://") {
					imageExt := cryptil.Md5Crypt(imageUrl) + filepath.Ext(imageUrl)

					dstFile := filepath.Join(baseDir, "uploads", time.Now().Format("200601"), imageExt)

					if err := requests.DownloadAndSaveFile(imageUrl, dstFile); err == nil {
						imageUrl = strings.TrimPrefix(strings.Replace(dstFile, "\\", "/", -1), strings.Replace(baseDir, "\\", "/", -1))
						if !strings.HasPrefix(imageUrl, "/") && !strings.HasPrefix(imageUrl, "\\") {
							imageUrl = "/" + imageUrl
						}
					}
				} else if strings.HasPrefix(imageUrl, "/") {
					filetil.CopyFile(filepath.Join(config.WorkingDirectory, imageUrl), filepath.Join(baseDir, imageUrl))
				}

				imageUrl = strings.Replace(strings.TrimSuffix(image, originalImageUrl+")")+imageUrl+")", "\\", "/", -1)

				return imageUrl
			})

			linkRe := regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)

			markdown = linkRe.ReplaceAllStringFunc(markdown, func(link string) string {
				links := linkRe.FindAllStringSubmatch(link, -1)
				if len(links) > 0 && len(links[0]) >= 3 {
					originalLink := links[0][2]

					//如果当前链接位于当前项目内
					if strings.HasPrefix(originalLink, bookUrl) {
						docIdentify := strings.TrimSpace(strings.TrimPrefix(originalLink, bookUrl))
						tempDoc := model.NewDocument()
						if id, err := strconv.Atoi(docIdentify); err == nil && id > 0 {
							err := o.QueryTable(model.NewDocument().TableNameWithPrefix()).Filter("document_id", id).One(tempDoc, "identify", "parent_id", "document_id")
							if err != nil {
								logs.Error(err)
								return link
							}
						} else {
							err := o.QueryTable(model.NewDocument().TableNameWithPrefix()).Filter("identify", docIdentify).One(tempDoc, "identify", "parent_id", "document_id")
							if err != nil {
								logs.Error(err)
								return link
							}
						}
						tempLink := recursiveJoinDocumentIdentify(tempDoc.ParentId, "") + strings.TrimPrefix(originalLink, bookUrl)

						if !strings.HasSuffix(tempLink, ".md") && !strings.HasSuffix(doc.Identify, ".markdown") {
							tempLink = tempLink + ".md"
						}
						relative := strings.TrimPrefix(strings.Replace(p, "\\", "/", -1), strings.Replace(baseDir, "\\", "/", -1))
						repeat := 0
						if relative != "" {
							relative = strings.TrimSuffix(strings.TrimPrefix(relative, "/"), "/")
							repeat = strings.Count(relative, "/") + 1
						}
						logs.Info(repeat, "|", relative, "|", p, "|", baseDir)
						tempLink = strings.Repeat("../", repeat) + tempLink

						link = strings.TrimSuffix(link, originalLink+")") + tempLink + ")"
					}
				}

				return link

			})
		} else {
			markdown = "# " + doc.DocumentName + "\n"
		}
		if err := os.WriteFile(docPath, []byte(markdown), 0644); err != nil {
			logs.Error("导出Markdown失败->", err)
			return err
		}

		if subDocCount > 0 {
			if err = exportMarkdown(dirPath, doc.DocumentId, bookId, baseDir, bookUrl); err != nil {
				return err
			}
		}
	}
	return nil
}

func recursiveJoinDocumentIdentify(parentDocId int, identify string) string {
	o := orm.NewOrm()

	doc := model.NewDocument()

	err := o.QueryTable(model.NewDocument().TableNameWithPrefix()).Filter("document_id", parentDocId).One(doc, "identify", "parent_id", "document_id")

	if err != nil {
		logs.Error(err)
		return identify
	}

	if doc.Identify == "" {
		identify = strconv.Itoa(doc.DocumentId) + "/" + identify
	} else {
		identify = doc.Identify + "/" + identify
	}
	if doc.ParentId > 0 {
		identify = recursiveJoinDocumentIdentify(doc.ParentId, identify)
	}
	return identify
}
