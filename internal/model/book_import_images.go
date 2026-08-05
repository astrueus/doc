package model

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/pkg/cryptil"
	"git.itopcms.com/astrueus/doc/pkg/filetil"
	"git.itopcms.com/astrueus/doc/pkg/requests"
	"github.com/beego/beego/v2/core/logs"
)

// rewriteImportMarkdownImage rewrites a Markdown image token during ImportBook.
func rewriteImportMarkdownImage(image, path, tempPath string, re *regexp.Regexp) string {
	images := re.FindAllSubmatch([]byte(image), -1)
	if len(images) <= 0 || len(images[0]) < 3 {
		return image
	}
	originalImageUrl := string(images[0][2])
	imageUrl := strings.Replace(string(originalImageUrl), "\\", "/", -1)

	//本地路径：复制到项目 uploads；远程路径：下载后替换
	if !strings.HasPrefix(imageUrl, "http://") &&
		!strings.HasPrefix(imageUrl, "https://") &&
		!strings.HasPrefix(imageUrl, "ftp://") {
		if l := strings.Index(imageUrl, "?"); l > 0 {
			imageUrl = imageUrl[:l]
		}

		if strings.HasPrefix(imageUrl, "/") {
			imageUrl = filepath.Join(tempPath, imageUrl)
		} else if strings.HasPrefix(imageUrl, "./") {
			imageUrl = filepath.Join(filepath.Dir(path), strings.TrimPrefix(imageUrl, "./"))
		} else if strings.HasPrefix(imageUrl, "../") {
			imageUrl = filepath.Join(filepath.Dir(path), imageUrl)
		} else {
			imageUrl = filepath.Join(filepath.Dir(path), imageUrl)
		}
		imageUrl = strings.Replace(imageUrl, "\\", "/", -1)
		dstFile := filepath.Join(config.WorkingDirectory, "uploads", time.Now().Format("200601"), strings.TrimPrefix(imageUrl, tempPath))

		if filetil.FileExists(imageUrl) {
			filetil.CopyFile(imageUrl, dstFile)

			imageUrl = strings.TrimPrefix(strings.Replace(dstFile, "\\", "/", -1), strings.Replace(config.WorkingDirectory, "\\", "/", -1))

			if !strings.HasPrefix(imageUrl, "/") && !strings.HasPrefix(imageUrl, "\\") {
				imageUrl = "/" + imageUrl
			}
		}

	} else {
		imageExt := cryptil.Md5Crypt(imageUrl) + filepath.Ext(imageUrl)

		dstFile := filepath.Join(config.WorkingDirectory, "uploads", time.Now().Format("200601"), imageExt)

		if err := requests.DownloadAndSaveFile(imageUrl, dstFile); err == nil {
			imageUrl = strings.TrimPrefix(strings.Replace(dstFile, "\\", "/", -1), strings.Replace(config.WorkingDirectory, "\\", "/", -1))
			if !strings.HasPrefix(imageUrl, "/") && !strings.HasPrefix(imageUrl, "\\") {
				imageUrl = "/" + imageUrl
			}
		}
	}

	imageUrl = strings.Replace(strings.TrimSuffix(image, originalImageUrl+")")+config.URLForWithCdnImage(imageUrl)+")", "\\", "/", -1)
	return imageUrl
}

// rewriteImportMarkdownLink rewrites a Markdown link token during ImportBook.
func rewriteImportMarkdownLink(link, path, tempPath, bookIdentify string, linkRegexp *regexp.Regexp) string {
	links := linkRegexp.FindAllStringSubmatch(link, -1)
	originalLink := links[0][2]
	var linkPath string
	var err error
	if strings.HasPrefix(originalLink, "<") {
		originalLink = strings.TrimPrefix(originalLink, "<")
	}
	if strings.HasSuffix(originalLink, ">") {
		originalLink = strings.TrimSuffix(originalLink, ">")
	}
	if strings.HasPrefix(originalLink, "/") {
		linkPath, err = filepath.Abs(filepath.Join(tempPath, originalLink))
	} else if strings.HasPrefix(originalLink, "./") {
		linkPath, err = filepath.Abs(filepath.Join(filepath.Dir(path), originalLink[1:]))
	} else {
		linkPath, err = filepath.Abs(filepath.Join(filepath.Dir(path), originalLink))
	}

	if err == nil {
		if filetil.FileExists(linkPath) {
			ext := filepath.Ext(linkPath)
			if strings.EqualFold(ext, ".md") || strings.EqualFold(ext, ".markdown") {
				docIdentify := strings.Replace(strings.TrimPrefix(strings.Replace(linkPath, "\\", "/", -1), tempPath+"/"), "/", "-", -1)
				if ok, err := regexp.MatchString(`[a-z]+[a-zA-Z0-9_.\-]*$`, docIdentify); !ok || err != nil {
					docIdentify = "import-" + docIdentify
				}
				docIdentify = strings.TrimSuffix(docIdentify, "-README.md")

				link = strings.TrimSuffix(link, originalLink+")") + config.URLFor("DocumentController.Read", ":key", bookIdentify, ":id", docIdentify) + ")"

			} else {
				dstPath := filepath.Join(config.WorkingDirectory, "uploads", time.Now().Format("200601"), originalLink)

				filetil.CopyFile(linkPath, dstPath)

				tempLink := config.BaseUrl + strings.TrimPrefix(strings.Replace(dstPath, "\\", "/", -1), strings.Replace(config.WorkingDirectory, "\\", "/", -1))

				link = strings.TrimSuffix(link, originalLink+")") + tempLink + ")"

			}
		} else {
			logs.Info("文件不存在 ->", linkPath)
		}
	}

	return link
}
