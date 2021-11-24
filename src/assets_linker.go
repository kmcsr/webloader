
package webopener

import (
	io "io"
	os "os"
	filex "github.com/kmcsr/go-util/file"
)

type AssetsLoadHandle func(l *AssetsLinker, src string, w *io.WriteCloser)

type AssetsLinker struct{
	assets_src, assets_dst string
	link_map map[string]string
	handles []AssetsLoadHandle
}

func NewAssetsLinker(assets_src, assets_dst string)(*AssetsLinker){
	return &AssetsLinker{
		assets_src: assets_src,
		assets_dst: assets_dst,
		link_map: make(map[string]string),
		handles: make([]AssetsLoadHandle, 0),
	}
}

func (l *AssetsLinker)GetHandles()([]AssetsLoadHandle){
	return l.handles
}

func (l *AssetsLinker)AddHandle(f ...AssetsLoadHandle)([]AssetsLoadHandle){
	l.handles = append(l.handles, f...)
	return f
}

func (l *AssetsLinker)SetHandle(f ...AssetsLoadHandle)([]AssetsLoadHandle){
	l.handles = make([]AssetsLoadHandle, len(f))
	copy(l.handles, f)
	return f
}

func (l *AssetsLinker)Load()(err error){
	l.link_map = make(map[string]string)
	err = filex.Walk(l.assets_src, func(e *filex.WalkEnity, err error)(error){
		if err != nil { return err }
		if e.IsDir() {
			err = filex.MakeDir(filex.JoinPath(l.assets_dst, e.Path()), 0744)
		}else{
			var (
				sfd *os.File
				writer io.WriteCloser
				mode os.FileMode = os.ModePerm
				hs string
			)
			hs, err = calculateFileHash(e.FullPath())
			if err != nil { return err }
			b, s := filex.SplitNameL(e.Name())
			p := filex.JoinPath(e.ParentPath(), b + "@" + hs + s)
			l.link_map[e.Path()] = p
			// copy
			sfd, err = os.Open(e.FullPath())
			if err != nil { return err }
			defer sfd.Close()
			if info, e := sfd.Stat(); e == nil {
				mode = info.Mode()
			}
			writer, err = os.OpenFile(filex.JoinPath(l.assets_dst, p), os.O_WRONLY | os.O_CREATE | os.O_TRUNC, mode)
			if err != nil { return err }
			defer writer.Close()
			for _, c := range l.handles {
				c(l, e.FullPath(), &writer)
			}
			_, err = io.Copy(writer, sfd)
		}
		return err
	})
	if err != nil { return err }
	return nil
}

func (l *AssetsLinker)GetAssetPath(path string)(v string, ok bool){
	v, ok = l.link_map[path]
	return
}
