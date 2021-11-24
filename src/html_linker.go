
package webopener

import (
	os "os"
	unicode "unicode"
	strings "strings"

	gquery "github.com/kmcsr/gquery"
	filex "github.com/kmcsr/go-util/file"
)

type HtmlLoadHandle func(l *HtmlLinker, src string, nd gquery.Node)

type htmlHandleMapItem struct{
	h HtmlLoadHandle
	d bool
}

var htmlLoadHandleMap = make(map[string]*htmlHandleMapItem)
var htmlLoadHandleAlias = make(map[string]string)

func RegisterHtmlLoadHandle(id string, def bool, handle HtmlLoadHandle, aliases ...string){
	htmlLoadHandleMap[id] = &htmlHandleMapItem{
		h: handle,
		d: def,
	}
	for _, v := range aliases {
		htmlLoadHandleAlias[v] = id
	}
}

func GetHtmlLoadHandle(key string)(HtmlLoadHandle, bool){
	h, ok := htmlLoadHandleMap[key]
	if !ok {
		if key, ok = htmlLoadHandleAlias[key]; ok {
			h, _ = htmlLoadHandleMap[key]
		}
	}
	if h == nil {
		return nil, false
	}
	return h.h, h.d
}

func GetHtmlLoadHandleId(key string)(_ string, ok bool){
	if _, ok = htmlLoadHandleMap[key]; !ok {
		key, ok = htmlLoadHandleAlias[key]
	}
	return key, ok
}

func init(){
	RegisterHtmlLoadHandle("zip", true, func(hl *HtmlLinker, _ string, n gquery.Node){
		switch n.Name() {
		case "#text":
			v := zipString(n.GetValue())
			if gquery.IsBlockNodeName(n.Name()) {
				v = strings.TrimSpace(v)
			}else{
				if len(v) > 0 && v[0] == ' ' {
					if before := gquery.FindPrevNodeExcepts(n, "#comment", "script", "style",
						"br", "hr", "meta", "link", "input", "img"); before != nil {
						if bv := ([]rune)(before.GetText()); len(bv) > 0 && unicode.IsSpace(bv[len(bv) - 1]) {
							v = v[1:]
						}
					}else{
						v = v[1:]
					}
				}
				if len(v) > 0 && v[len(v) - 1] == ' ' && n.After() == nil {
					v = v[:len(v) - 1]
				}
			}
			n.SetValue(v)
		case "script":
			n.SetValue(zipCodeJs(n.GetValue()))
		case "style":
			n.SetValue(zipCodeCss(n.GetValue()))
		}
		if n.HasAttr("style") {
			n.SetAttr("style", zipCodeCss(n.GetAttr("style")))
		}
	}, "allow-zip")
	RegisterHtmlLoadHandle("no-comment", true, func(hl *HtmlLinker, _ string, n gquery.Node){
		if hl.GetHandleStatus("zip") && n.Name() == "#comment" {
			n.Parent().RemoveChild(n)
		}
	}, "disallow-comment")
	RegisterHtmlLoadHandle("link-assets", true, func(hl *HtmlLinker, _ string, n gquery.Node){
		var aid string
		switch n.Name() {
		case "a", "link":
			aid = "href"
		case "img", "script":
			aid = "src"
		default: return
		}
		if n.HasAttr(aid) {
			src := n.GetAttr(aid)
			if strings.HasPrefix(src, "@/") {
				if dst, ok := hl.assets_linker.GetAssetPath(src[2:]); ok {
					n.SetAttr(aid, hl.assets_prefix + dst)
				}
			}
		}
	}, "allow-link-assets")
}

type HtmlLinker struct{
	html_src, html_dst string
	assets_linker *AssetsLinker
	assets_prefix string
	handle_map map[string]bool
}

func NewHtmlLinker(html_src, html_dst string, allows ...string)(hl *HtmlLinker){
	handle_map := make(map[string]bool)
	for k, v := range htmlLoadHandleMap {
		handle_map[k] = v.d
	}
	for _, k := range allows {
		if len(k) > 1 && k[0] == '!' {
			if k, ok := GetHtmlLoadHandleId(k[1:]); ok {
				handle_map[k] = false
			}
		}else{
			if k, ok := GetHtmlLoadHandleId(k); ok {
				handle_map[k] = true
			}
		}
	}
	return &HtmlLinker{
		html_src: html_src,
		html_dst: html_dst,
		assets_linker: nil,
		assets_prefix: "",
		handle_map: handle_map,
	}
}

func (hl *HtmlLinker)GetHandleStatus(key string)(bool){
	s, ok := hl.handle_map[key]
	return ok && s
}

func (hl *HtmlLinker)SetHandleStatus(key string, s bool)(*HtmlLinker){
	hl.handle_map[key] = s
	return hl
}

func (hl *HtmlLinker)SetAssetsLinker(al *AssetsLinker)(*HtmlLinker){
	hl.assets_linker = al
	return hl
}

func (hl *HtmlLinker)GetAssetsLinker()(*AssetsLinker){
	return hl.assets_linker
}

func (hl *HtmlLinker)SetAssetsPrefix(prefix string)(*HtmlLinker){
	if len(prefix) > 0 && prefix[len(prefix) - 1] != '/' {
		prefix += "/"
	}
	hl.assets_prefix = prefix
	return hl
}

func (hl *HtmlLinker)GetAssetsPrefix()(string){
	return hl.assets_prefix
}

func (hl *HtmlLinker)fixDoc(src string, doc *gquery.Document){
	oldmap := make(map[string]bool)
	defer func(){
		for k, v := range oldmap {
			hl.handle_map[k] = v
		}
	}()
	doc.GetNodeList().ForEach(func(n gquery.Node, _ int){
		if n.Name() == "#comment" {
			var ok bool
			content := n.GetValue()
			if len(content) > 0 && content[0] == '$' {
				lines := strings.Split(content[1:], ";")
				for _, l := range lines {
					si := strings.IndexByte(l, ':')
					if si == -1 { break }
					key, value := strings.TrimSpace(l[:si]), strings.TrimSpace(l[si + 1:])
					if key, ok = GetHtmlLoadHandleId(key); ok {
						oldmap[key] = hl.handle_map[key]
						hl.handle_map[key] = strToBool(value)
					}
				}
			}
		}
	})
	calls := make([]HtmlLoadHandle, 0, 3)
	for k, v := range hl.handle_map {
		if v {
			if h, ok := htmlLoadHandleMap[k]; ok {
				calls = append(calls, h.h)
			}
		}
	}
	doc.GetHtmlNode().IterAllChildren(func(n gquery.Node){
		for _, c := range calls {
			c(hl, src, n)
		}
	})
}

func (hl *HtmlLinker)Load()(err error){
	if hl.assets_linker != nil {
		err = hl.assets_linker.Load()
		if err != nil { return }
	}
	err = filex.Walk(hl.html_src, func(e *filex.WalkEnity, err error)(error){
		if err != nil {
			return err
		}
		if e.IsDir() {
			err = filex.MakeDir(filex.JoinPath(hl.html_dst, e.Path()), 0744)
		}else if filex.HasSuffix(e.Name(), ".html", ".htm") {
			var (
				fd *os.File
				doc *gquery.Document
			)
			doc, err = gquery.DecodeFile(e.FullPath())
			if err != nil { return err }
			hl.fixDoc(e.FullPath(), doc)
			fd, err = os.Create(filex.JoinPath(hl.html_dst, e.Path()))
			if err != nil { return err }
			defer fd.Close()
			_, err = doc.WriteTo(fd)
		}
		if err != nil { return err }
		return nil
	})
	if err != nil {
		return
	}
	return
}
