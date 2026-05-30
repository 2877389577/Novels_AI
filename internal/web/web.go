package web

import (
	"embed"
	"io/fs"
	"net/http"
)

// embeddedFiles 在编译期把 dist 目录嵌入二进制。
// 用 all: 前缀确保以 . 或 _ 开头的文件也被嵌入，避免未来新增此类静态文件被静默丢弃。
//
//go:embed all:dist
var embeddedFiles embed.FS

// distFS 根为 dist；assetsFS 根为 dist/assets。
// 在包初始化时剥离前缀——dist 由 //go:embed 编译期保证存在，err 必为 nil，可安全忽略。
var (
	distFS, _   = fs.Sub(embeddedFiles, "dist")
	assetsFS, _ = fs.Sub(embeddedFiles, "dist/assets")
)

// DistFS 返回根为 dist 的文件系统，用于 index.html、favicon.ico 等顶层文件。
func DistFS() http.FileSystem { return http.FS(distFS) }

// AssetsFS 返回根为 dist/assets 的文件系统，用于 /assets/* 静态资源。
// 单独剥掉 assets 前缀，是因为 gin.StaticFS("/assets", fs) 内部会用
// http.StripPrefix("/assets", ...)，要求传入的 fs 根正好是 assets 目录，否则 404。
func AssetsFS() http.FileSystem { return http.FS(assetsFS) }

// indexHTML 是 dist/index.html 的内容，在包初始化时一次性读出。
// 单独提供字节、而不用 http.FileServer 托管根路径，是为了避开 net/http 对
// /index.html 的 301 重定向（→ ./），否则会与根路由 "/" 形成无限重定向。
var indexHTML, _ = embeddedFiles.ReadFile("dist/index.html")

// IndexHTML 返回前端入口 index.html 的内容，供根路由与 SPA history 回退直接写回。
func IndexHTML() []byte { return indexHTML }
